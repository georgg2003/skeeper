// Package usecase implements the CLI application layer: vault crypto, entry CRUD, auth, and sync orchestration.
package usecase

//go:generate go tool mockgen -typed -destination=mock_secret_test.go -package=usecase -source=secret.go LocalSecretStore,SessionReader,VaultRemote

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/georgg2003/skeeper/internal/client/pkg/crypto"
	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	skeeperremote "github.com/georgg2003/skeeper/internal/client/repository/skeeper"
)

var (
	// ErrWrongMasterPassword is returned when the derived key does not match the stored verifier.
	ErrWrongMasterPassword = errors.New("wrong master password")
	// ErrEntryNotFound is returned when no row exists for the UUID (and user).
	ErrEntryNotFound = errors.New("entry not found")
	// ErrEntryDeleted is returned for soft-deleted rows.
	ErrEntryDeleted = errors.New("entry was deleted")
	// ErrWrongEntryType is returned when an update/delete operation does not match the stored type.
	ErrWrongEntryType = errors.New("entry type does not match this operation")
)

// DefaultMaxFileBytes is the FILE payload cap when config does not set a positive limit.
const DefaultMaxFileBytes = 10 << 20

// EntryMetadata is cleartext JSON encrypted with the entry DEK alongside the payload.
type EntryMetadata struct {
	Name             string            `json:"name"`
	Notes            string            `json:"notes,omitempty"`
	OriginalFilename string            `json:"original_filename,omitempty"` // FILE entries: basename for `get` output
	ExtraTags        map[string]string `json:"tags,omitempty"`
}

// LocalSecretStore abstracts the SQLite vault: entries, dirty flags, and vault KDF salt/verifier.
type LocalSecretStore interface {
	GetDirtyEntries(ctx context.Context, forUserID *int64) ([]models.Entry, error)
	MarkAsSynced(ctx context.Context, id uuid.UUID, userID int64) error
	SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error
	GetLastUpdate(ctx context.Context, forUserID *int64) (time.Time, error)
	GetEntry(ctx context.Context, id uuid.UUID, forUserID *int64) (models.Entry, error)
	EnsureLocalVaultCrypto(ctx context.Context, userID int64) (salt []byte, masterVerifier []byte, err error)
	ReplaceLocalVaultCrypto(ctx context.Context, userID int64, salt, masterVerifier []byte) error
	SetLocalMasterVerifier(ctx context.Context, userID int64, masterVerifier []byte) error
	ListEntries(ctx context.Context, forUserID *int64) ([]models.Entry, error)
}

// SessionReader reads the persisted Auther session (for user id and auth state).
type SessionReader interface {
	GetSession(ctx context.Context) (*models.Session, error)
}

// VaultRemote publishes or fetches KDF salt and master-key verifier for the logged-in user.
type VaultRemote interface {
	GetVaultCrypto(ctx context.Context) (kdfSalt, masterVerifier []byte, err error)
	PutVaultCrypto(ctx context.Context, kdfSalt, masterVerifier []byte) error
}

// SecretUseCase encrypts and decrypts vault entries on the client and persists them locally.
type SecretUseCase struct {
	local        LocalSecretStore
	sessions     SessionReader
	remote       VaultRemote
	log          *slog.Logger
	maxFileBytes int64
}

// NewSecretUseCase builds a vault use case. Remote may be nil if vault crypto is never uploaded.
// maxFileBytes <= 0 uses DefaultMaxFileBytes for FILE payloads.
func NewSecretUseCase(local LocalSecretStore, sessions SessionReader, remote VaultRemote, log *slog.Logger, maxFileBytes int64) *SecretUseCase {
	if maxFileBytes <= 0 {
		maxFileBytes = DefaultMaxFileBytes
	}
	return &SecretUseCase{
		local:        local,
		sessions:     sessions,
		remote:       remote,
		log:          log.With("component", "secret_usecase"),
		maxFileBytes: maxFileBytes,
	}
}

func (*SecretUseCase) sanitizeOriginalFilename(name string) string {
	base := filepath.Base(name)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "file"
	}
	return base
}

// entrySeal attaches vault crypto operations to *models.Entry only (pkg/models stays free of crypto imports).
type entrySeal struct {
	e *models.Entry
}

func newEntrySeal(e *models.Entry) *entrySeal {
	if e == nil {
		panic("entrySeal: nil *models.Entry")
	}
	return &entrySeal{e: e}
}

func (uc *SecretUseCase) activeAutherUserID(ctx context.Context) *int64 {
	s, err := uc.sessions.GetSession(ctx)
	if err != nil || s == nil || s.UserID == nil {
		return nil
	}
	return s.UserID
}

func (uc *SecretUseCase) requireAutherUserID(ctx context.Context) (int64, error) {
	s, err := uc.sessions.GetSession(ctx)
	if err != nil || s == nil || s.UserID == nil {
		return 0, errors.New("log in required")
	}
	return *s.UserID, nil
}

func (uc *SecretUseCase) pullRemoteVaultCrypto(ctx context.Context, userID int64) error {
	if uc.remote == nil {
		return nil
	}
	salt, verifier, err := uc.remote.GetVaultCrypto(ctx)
	if err != nil {
		if errors.Is(err, skeeperremote.ErrVaultCryptoNotFound) {
			return nil
		}
		return fmt.Errorf("fetch vault crypto: %w", err)
	}
	if len(salt) == 0 || len(verifier) == 0 {
		return nil
	}
	return uc.local.ReplaceLocalVaultCrypto(ctx, userID, salt, verifier)
}

func (uc *SecretUseCase) materializeVaultCrypto(ctx context.Context) (salt []byte, verifier []byte, err error) {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := uc.pullRemoteVaultCrypto(ctx, uid); err != nil {
		return nil, nil, err
	}
	return uc.local.EnsureLocalVaultCrypto(ctx, uid)
}

func (uc *SecretUseCase) deriveAndCheckMasterKey(salt, storedVerifier []byte, masterPass string) ([]byte, error) {
	masterKey := crypto.DeriveMasterKey(masterPass, salt)
	if len(storedVerifier) > 0 {
		v := crypto.MasterKeyVerifier(masterKey)
		if subtle.ConstantTimeCompare(v, storedVerifier) != 1 {
			return nil, ErrWrongMasterPassword
		}
	}
	return masterKey, nil
}

func (uc *SecretUseCase) publishVaultCrypto(ctx context.Context, salt, masterKey []byte) error {
	if uc.remote == nil || uc.activeAutherUserID(ctx) == nil {
		return nil
	}
	ver := crypto.MasterKeyVerifier(masterKey)
	return uc.remote.PutVaultCrypto(ctx, salt, ver)
}

// sealWithNewDEK encrypts plaintext + metadata with a fresh DEK, wraps the DEK with masterKey, writes ciphertext into s.e,
// bumps version, and sets updated-at / user id (caller saves). For a new row, start with Version 0 so the bump yields 1.
func (s *entrySeal) sealWithNewDEK(plaintext []byte, meta EntryMetadata, masterKey []byte, uid int64) error {
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return fmt.Errorf("generate dek: %w", err)
	}
	encPayload, err := crypto.EncryptAESGCM(plaintext, dek)
	if err != nil {
		return fmt.Errorf("encrypt payload: %w", err)
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	encMeta, err := crypto.EncryptAESGCM(metaBytes, dek)
	if err != nil {
		return fmt.Errorf("encrypt meta: %w", err)
	}
	encDEK, err := crypto.EncryptAESGCM(dek, masterKey)
	if err != nil {
		return fmt.Errorf("encrypt dek: %w", err)
	}
	s.e.Payload = encPayload
	s.e.Meta = encMeta
	s.e.EncryptedDek = encDEK
	s.e.Version++
	s.e.UpdatedAt = time.Now()
	s.e.UserID = &uid
	return nil
}

func (s *entrySeal) verifyPayloadAndMetaWithDEK(dek []byte) error {
	if _, err := crypto.DecryptAESGCM(s.e.Payload, dek); err != nil {
		return fmt.Errorf("decrypt payload: %w", err)
	}
	if _, err := crypto.DecryptAESGCM(s.e.Meta, dek); err != nil {
		return fmt.Errorf("decrypt meta: %w", err)
	}
	return nil
}

func (s *entrySeal) decryptMetaWithDEK(dek []byte) (EntryMetadata, error) {
	var out EntryMetadata
	metaBytes, err := crypto.DecryptAESGCM(s.e.Meta, dek)
	if err != nil {
		return out, fmt.Errorf("decrypt meta: %w", err)
	}
	if err := json.Unmarshal(metaBytes, &out); err != nil {
		return out, fmt.Errorf("unmarshal meta: %w", err)
	}
	return out, nil
}

func (s *entrySeal) encryptMetaOnly(meta EntryMetadata, dek []byte, uid int64) error {
	outMeta, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	encMeta, err := crypto.EncryptAESGCM(outMeta, dek)
	if err != nil {
		return fmt.Errorf("encrypt meta: %w", err)
	}
	s.e.Meta = encMeta
	s.e.Version++
	s.e.UpdatedAt = time.Now()
	s.e.UserID = &uid
	return nil
}

func (uc *SecretUseCase) insertEncryptedEntry(ctx context.Context, typ string, meta EntryMetadata, plaintext []byte, masterPass string) error {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return err
	}
	uc.log.Info("creating encrypted entry", "type", typ, "name", meta.Name, "payload_bytes", len(plaintext))

	salt, storedVerifier, err := uc.materializeVaultCrypto(ctx)
	if err != nil {
		return err
	}
	masterKey, err := uc.deriveAndCheckMasterKey(salt, storedVerifier, masterPass)
	if err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("new entry id: %w", err)
	}
	entry := models.Entry{
		UUID:      id,
		Type:      typ,
		IsDeleted: false,
	}
	if err := newEntrySeal(&entry).sealWithNewDEK(plaintext, meta, masterKey, uid); err != nil {
		return err
	}
	if err := uc.local.SaveEntry(ctx, entry, true); err != nil {
		return err
	}
	return uc.finalizeVaultVerifier(ctx, uid, salt, storedVerifier, masterKey)
}

func (uc *SecretUseCase) updateEntryReplaceContent(ctx context.Context, id uuid.UUID, wantType string, meta EntryMetadata, plaintext []byte, masterPass string) error {
	e, uid, masterKey, dek, err := uc.unlockEntry(ctx, id, masterPass)
	if err != nil {
		return err
	}
	if e.Type != wantType {
		return ErrWrongEntryType
	}
	seal := newEntrySeal(&e)
	if err := seal.verifyPayloadAndMetaWithDEK(dek); err != nil {
		return err
	}
	if err := seal.sealWithNewDEK(plaintext, meta, masterKey, uid); err != nil {
		return err
	}
	return uc.local.SaveEntry(ctx, e, true)
}

// SetPassword stores a new PASSWORD entry.
func (uc *SecretUseCase) SetPassword(ctx context.Context, meta EntryMetadata, password string, masterPass string) error {
	return uc.insertEncryptedEntry(ctx, models.EntryTypePassword, meta, []byte(password), masterPass)
}

// SetText stores a new TEXT entry.
func (uc *SecretUseCase) SetText(ctx context.Context, meta EntryMetadata, text string, masterPass string) error {
	return uc.insertEncryptedEntry(ctx, models.EntryTypeText, meta, []byte(text), masterPass)
}

// SetFile encrypts file bytes into entry.Payload (same as other entry types) so sync persists ciphertext in Postgres.
func (uc *SecretUseCase) SetFile(ctx context.Context, meta EntryMetadata, originalFilename string, data []byte, masterPass string) error {
	if uc.maxFileBytes > 0 && int64(len(data)) > uc.maxFileBytes {
		return fmt.Errorf("file too large (%d bytes); max is %d bytes", len(data), uc.maxFileBytes)
	}
	metaForStore := meta
	metaForStore.OriginalFilename = uc.sanitizeOriginalFilename(originalFilename)
	return uc.insertEncryptedEntry(ctx, models.EntryTypeFile, metaForStore, data, masterPass)
}

// SetCard stores a new CARD entry.
func (uc *SecretUseCase) SetCard(ctx context.Context, meta EntryMetadata, card models.CardPayload, masterPass string) error {
	payload, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}
	return uc.insertEncryptedEntry(ctx, models.EntryTypeCard, meta, payload, masterPass)
}

func (uc *SecretUseCase) finalizeVaultVerifier(ctx context.Context, userID int64, salt, storedVerifier []byte, masterKey []byte) error {
	if len(storedVerifier) == 0 {
		ver := crypto.MasterKeyVerifier(masterKey)
		if err := uc.local.SetLocalMasterVerifier(ctx, userID, ver); err != nil {
			return fmt.Errorf("save master verifier: %w", err)
		}
	}
	return uc.publishVaultCrypto(ctx, salt, masterKey)
}

// ListLocal returns ciphertext entries for the current session’s user.
func (uc *SecretUseCase) ListLocal(ctx context.Context) ([]models.Entry, error) {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.local.ListEntries(ctx, &uid)
}

// GetLocalEntry loads one ciphertext row by UUID or returns ErrEntryNotFound.
func (uc *SecretUseCase) GetLocalEntry(ctx context.Context, id uuid.UUID) (models.Entry, error) {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return models.Entry{}, err
	}
	e, err := uc.local.GetEntry(ctx, id, &uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Entry{}, ErrEntryNotFound
		}
		return models.Entry{}, err
	}
	return e, nil
}

// unlockEntry loads a non-deleted entry and unwraps its DEK using the vault master password.
func (uc *SecretUseCase) unlockEntry(ctx context.Context, id uuid.UUID, masterPass string) (e models.Entry, uid int64, masterKey, dek []byte, err error) {
	uid, err = uc.requireAutherUserID(ctx)
	if err != nil {
		return
	}
	e, err = uc.local.GetEntry(ctx, id, &uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrEntryNotFound
		}
		return
	}
	if e.IsDeleted {
		err = ErrEntryDeleted
		return
	}
	var salt, verifier []byte
	salt, verifier, err = uc.materializeVaultCrypto(ctx)
	if err != nil {
		return
	}
	masterKey, err = uc.deriveAndCheckMasterKey(salt, verifier, masterPass)
	if err != nil {
		return
	}
	dek, err = crypto.DecryptAESGCM(e.EncryptedDek, masterKey)
	if err != nil {
		err = fmt.Errorf("decrypt entry dek: %w", err)
	}
	return
}

// DeleteEntry soft-deletes an entry (syncs as is_deleted); master password proves vault access.
func (uc *SecretUseCase) DeleteEntry(ctx context.Context, id uuid.UUID, masterPass string) error {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return err
	}
	e, err := uc.local.GetEntry(ctx, id, &uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEntryNotFound
		}
		return err
	}
	if e.IsDeleted {
		return ErrEntryDeleted
	}
	salt, verifier, err := uc.materializeVaultCrypto(ctx)
	if err != nil {
		return err
	}
	if _, err := uc.deriveAndCheckMasterKey(salt, verifier, masterPass); err != nil {
		return err
	}
	e.IsDeleted = true
	e.Version++
	e.UpdatedAt = time.Now()
	return uc.local.SaveEntry(ctx, e, true)
}

// UpdatePassword replaces the stored password and metadata; increments version for sync (last-write-wins on the server).
func (uc *SecretUseCase) UpdatePassword(ctx context.Context, id uuid.UUID, meta EntryMetadata, password string, masterPass string) error {
	return uc.updateEntryReplaceContent(ctx, id, models.EntryTypePassword, meta, []byte(password), masterPass)
}

// UpdateText replaces stored text and metadata.
func (uc *SecretUseCase) UpdateText(ctx context.Context, id uuid.UUID, meta EntryMetadata, text string, masterPass string) error {
	return uc.updateEntryReplaceContent(ctx, id, models.EntryTypeText, meta, []byte(text), masterPass)
}

// UpdateCard replaces card fields and metadata.
func (uc *SecretUseCase) UpdateCard(ctx context.Context, id uuid.UUID, meta EntryMetadata, card models.CardPayload, masterPass string) error {
	payload, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}
	return uc.updateEntryReplaceContent(ctx, id, models.EntryTypeCard, meta, payload, masterPass)
}

// UpdateFile changes metadata and optionally replaces file bytes. Conflict handling matches other types: the server keeps the
// higher version and stores one full ciphertext blob per entry—there is no binary merge of file contents.
func (uc *SecretUseCase) UpdateFile(ctx context.Context, id uuid.UUID, meta EntryMetadata, masterPass string, replacePayload bool, newFile []byte, newOrigName string) error {
	e, uid, masterKey, dek, err := uc.unlockEntry(ctx, id, masterPass)
	if err != nil {
		return err
	}
	if e.Type != models.EntryTypeFile {
		return ErrWrongEntryType
	}
	seal := newEntrySeal(&e)
	storedMeta, err := seal.decryptMetaWithDEK(dek)
	if err != nil {
		return err
	}
	storedMeta.Name = meta.Name
	storedMeta.Notes = meta.Notes
	if meta.ExtraTags != nil {
		storedMeta.ExtraTags = meta.ExtraTags
	}

	if !replacePayload {
		if _, err := crypto.DecryptAESGCM(e.Payload, dek); err != nil {
			return fmt.Errorf("decrypt payload: %w", err)
		}
		if err := seal.encryptMetaOnly(storedMeta, dek, uid); err != nil {
			return err
		}
		return uc.local.SaveEntry(ctx, e, true)
	}

	if uc.maxFileBytes > 0 && int64(len(newFile)) > uc.maxFileBytes {
		return fmt.Errorf("file too large (%d bytes); max is %d bytes", len(newFile), uc.maxFileBytes)
	}
	storedMeta.OriginalFilename = uc.sanitizeOriginalFilename(newOrigName)
	if err := seal.sealWithNewDEK(newFile, storedMeta, masterKey, uid); err != nil {
		return err
	}
	return uc.local.SaveEntry(ctx, e, true)
}

// decryptPayloadAndMeta decrypts ciphertext fields using the master password (same path for every entry type).
func (uc *SecretUseCase) decryptPayloadAndMeta(ctx context.Context, entry models.Entry, masterPass string) (payload []byte, meta EntryMetadata, err error) {
	salt, storedVerifier, err := uc.materializeVaultCrypto(ctx)
	if err != nil {
		return nil, meta, err
	}
	masterKey, err := uc.deriveAndCheckMasterKey(salt, storedVerifier, masterPass)
	if err != nil {
		return nil, meta, err
	}
	dek, err := crypto.DecryptAESGCM(entry.EncryptedDek, masterKey)
	if err != nil {
		return nil, meta, fmt.Errorf("decryption failed (wrong master pass?): %w", err)
	}
	payload, err = crypto.DecryptAESGCM(entry.Payload, dek)
	if err != nil {
		return nil, meta, fmt.Errorf("decrypt payload: %w", err)
	}
	metaBytes, err := crypto.DecryptAESGCM(entry.Meta, dek)
	if err != nil {
		return nil, meta, fmt.Errorf("decrypt meta: %w", err)
	}
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, meta, fmt.Errorf("unmarshal meta: %w", err)
	}
	return payload, meta, nil
}

// GetDecryptedEntry returns plaintext payload, decrypted metadata, and for FILE entries the original base filename.
func (uc *SecretUseCase) GetDecryptedEntry(ctx context.Context, id uuid.UUID, masterPass string) ([]byte, *EntryMetadata, string, error) {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return nil, nil, "", err
	}
	entry, err := uc.local.GetEntry(ctx, id, &uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, "", ErrEntryNotFound
		}
		return nil, nil, "", err
	}
	if entry.IsDeleted {
		return nil, nil, "", ErrEntryDeleted
	}
	payload, meta, err := uc.decryptPayloadAndMeta(ctx, entry, masterPass)
	if err != nil {
		return nil, nil, "", err
	}
	if entry.Type == models.EntryTypeFile {
		orig := meta.OriginalFilename
		if orig == "" {
			orig = "file"
		}
		return payload, &meta, orig, nil
	}
	return payload, &meta, "", nil
}
