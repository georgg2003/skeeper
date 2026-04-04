package usecase

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/georgg2003/skeeper/internal/client/pkg/crypto"
	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	skeeperremote "github.com/georgg2003/skeeper/internal/client/repository/skeeper"
)

// ErrWrongMasterPassword is returned when the derived key does not match the stored verifier.
var ErrWrongMasterPassword = errors.New("wrong master password")

// EntryMetadata is cleartext metadata encrypted with the entry DEK before storage.
type EntryMetadata struct {
	Name      string            `json:"name"`
	Notes     string            `json:"notes,omitempty"`
	ExtraTags map[string]string `json:"tags,omitempty"`
}

// LocalSecretStore is the persistence surface required for encrypted entries and vault crypto metadata.
type LocalSecretStore interface {
	GetDirtyEntries(ctx context.Context, forUserID *int64) ([]models.Entry, error)
	MarkAsSynced(ctx context.Context, id uuid.UUID) error
	SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error
	GetLastUpdate(ctx context.Context, forUserID *int64) (time.Time, error)
	GetEntry(ctx context.Context, id uuid.UUID, forUserID *int64) (models.Entry, error)
	EnsureLocalVaultCrypto(ctx context.Context) (salt []byte, masterVerifier []byte, err error)
	ReplaceLocalVaultCrypto(ctx context.Context, salt, masterVerifier []byte) error
	SetLocalMasterVerifier(ctx context.Context, masterVerifier []byte) error
	ListEntries(ctx context.Context, forUserID *int64) ([]models.Entry, error)
}

// SessionReader returns the persisted Auther session (for per-user local scoping).
type SessionReader interface {
	GetSession(ctx context.Context) (*models.Session, error)
}

// VaultRemote fetches and stores per-user KDF salt + master-key verifier on the Skeeper server.
type VaultRemote interface {
	GetVaultCrypto(ctx context.Context) (kdfSalt, masterVerifier []byte, err error)
	PutVaultCrypto(ctx context.Context, kdfSalt, masterVerifier []byte) error
}

// SecretUseCase creates and reads ciphertext entries protected by a user master password.
type SecretUseCase struct {
	local    LocalSecretStore
	sessions SessionReader
	remote   VaultRemote
	log      *slog.Logger
}

// NewSecretUseCase constructs a SecretUseCase. remote may be nil (offline-only vault profile).
func NewSecretUseCase(local LocalSecretStore, sessions SessionReader, remote VaultRemote, log *slog.Logger) *SecretUseCase {
	return &SecretUseCase{
		local:    local,
		sessions: sessions,
		remote:   remote,
		log:      log.With("component", "secret_usecase"),
	}
}

// activeAutherUserID is the logged-in account for local row scoping; nil if no session or legacy session row.
func (uc *SecretUseCase) activeAutherUserID(ctx context.Context) *int64 {
	s, err := uc.sessions.GetSession(ctx)
	if err != nil || s == nil || s.UserID == nil {
		return nil
	}
	return s.UserID
}

func (uc *SecretUseCase) pullRemoteVaultCrypto(ctx context.Context) error {
	if uc.remote == nil || uc.activeAutherUserID(ctx) == nil {
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
	return uc.local.ReplaceLocalVaultCrypto(ctx, salt, verifier)
}

func (uc *SecretUseCase) materializeVaultCrypto(ctx context.Context) (salt []byte, verifier []byte, err error) {
	if err := uc.pullRemoteVaultCrypto(ctx); err != nil {
		return nil, nil, err
	}
	return uc.local.EnsureLocalVaultCrypto(ctx)
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

// SetPassword stores a login/password pair as an encrypted PASSWORD-type entry.
func (uc *SecretUseCase) SetPassword(ctx context.Context, meta EntryMetadata, password string, masterPass string) error {
	uc.log.Info("creating new encrypted entry", "name", meta.Name)

	salt, storedVerifier, err := uc.materializeVaultCrypto(ctx)
	if err != nil {
		return err
	}

	masterKey, err := uc.deriveAndCheckMasterKey(salt, storedVerifier, masterPass)
	if err != nil {
		return err
	}

	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return fmt.Errorf("generate dek: %w", err)
	}

	encryptedPayload, err := crypto.EncryptAESGCM([]byte(password), dek)
	if err != nil {
		return fmt.Errorf("encrypt payload: %w", err)
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	encryptedMeta, err := crypto.EncryptAESGCM(metaBytes, dek)
	if err != nil {
		return fmt.Errorf("encrypt meta: %w", err)
	}

	encryptedDEK, err := crypto.EncryptAESGCM(dek, masterKey)
	if err != nil {
		return fmt.Errorf("encrypt dek: %w", err)
	}

	id := uuid.New()
	entry := models.Entry{
		UUID:         id,
		Type:         models.EntryTypePassword,
		Payload:      encryptedPayload,
		EncryptedDek: encryptedDEK,
		Meta:         encryptedMeta,
		Version:      1,
		UpdatedAt:    time.Now(),
		IsDeleted:    false,
		UserID:       uc.activeAutherUserID(ctx),
	}

	if err := uc.local.SaveEntry(ctx, entry, true); err != nil {
		return err
	}
	return uc.finalizeVaultVerifier(ctx, salt, storedVerifier, masterKey)
}

// SetText stores arbitrary cleartext as an encrypted TEXT-type entry.
func (uc *SecretUseCase) SetText(ctx context.Context, meta EntryMetadata, text string, masterPass string) error {
	return uc.setBlob(ctx, models.EntryTypeText, meta, []byte(text), masterPass)
}

// SetBinary stores arbitrary bytes as an encrypted BINARY-type entry.
func (uc *SecretUseCase) SetBinary(ctx context.Context, meta EntryMetadata, data []byte, masterPass string) error {
	return uc.setBlob(ctx, models.EntryTypeBinary, meta, data, masterPass)
}

// SetCard stores card fields as JSON inside an encrypted CARD-type entry.
func (uc *SecretUseCase) SetCard(ctx context.Context, meta EntryMetadata, card models.CardPayload, masterPass string) error {
	payload, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}
	return uc.setBlob(ctx, models.EntryTypeCard, meta, payload, masterPass)
}

func (uc *SecretUseCase) setBlob(ctx context.Context, typ string, meta EntryMetadata, plaintext []byte, masterPass string) error {
	uc.log.Info("creating encrypted entry", "type", typ, "name", meta.Name)

	salt, storedVerifier, err := uc.materializeVaultCrypto(ctx)
	if err != nil {
		return err
	}

	masterKey, err := uc.deriveAndCheckMasterKey(salt, storedVerifier, masterPass)
	if err != nil {
		return err
	}

	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return fmt.Errorf("generate dek: %w", err)
	}

	encryptedPayload, err := crypto.EncryptAESGCM(plaintext, dek)
	if err != nil {
		return fmt.Errorf("encrypt payload: %w", err)
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	encryptedMeta, err := crypto.EncryptAESGCM(metaBytes, dek)
	if err != nil {
		return fmt.Errorf("encrypt meta: %w", err)
	}

	encryptedDEK, err := crypto.EncryptAESGCM(dek, masterKey)
	if err != nil {
		return fmt.Errorf("encrypt dek: %w", err)
	}

	entry := models.Entry{
		UUID:         uuid.New(),
		Type:         typ,
		Payload:      encryptedPayload,
		EncryptedDek: encryptedDEK,
		Meta:         encryptedMeta,
		Version:      1,
		UpdatedAt:    time.Now(),
		IsDeleted:    false,
		UserID:       uc.activeAutherUserID(ctx),
	}

	if err := uc.local.SaveEntry(ctx, entry, true); err != nil {
		return err
	}
	return uc.finalizeVaultVerifier(ctx, salt, storedVerifier, masterKey)
}

func (uc *SecretUseCase) finalizeVaultVerifier(ctx context.Context, salt, storedVerifier []byte, masterKey []byte) error {
	if len(storedVerifier) == 0 {
		ver := crypto.MasterKeyVerifier(masterKey)
		if err := uc.local.SetLocalMasterVerifier(ctx, ver); err != nil {
			return fmt.Errorf("save master verifier: %w", err)
		}
	}
	return uc.publishVaultCrypto(ctx, salt, masterKey)
}

// ListLocal returns ciphertext rows for display of ids and types without decryption.
// When the local session has a user id, only that user's rows are listed.
func (uc *SecretUseCase) ListLocal(ctx context.Context) ([]models.Entry, error) {
	return uc.local.ListEntries(ctx, uc.activeAutherUserID(ctx))
}

// GetLocalEntry returns one ciphertext row (e.g. to read the type label before decrypting payload).
func (uc *SecretUseCase) GetLocalEntry(ctx context.Context, id uuid.UUID) (models.Entry, error) {
	return uc.local.GetEntry(ctx, id, uc.activeAutherUserID(ctx))
}

// GetDecryptedEntry decrypts a single entry after deriving the master key.
func (uc *SecretUseCase) GetDecryptedEntry(ctx context.Context, id uuid.UUID, masterPass string) ([]byte, *EntryMetadata, error) {
	entry, err := uc.local.GetEntry(ctx, id, uc.activeAutherUserID(ctx))
	if err != nil {
		return nil, nil, err
	}

	salt, storedVerifier, err := uc.materializeVaultCrypto(ctx)
	if err != nil {
		return nil, nil, err
	}

	masterKey, err := uc.deriveAndCheckMasterKey(salt, storedVerifier, masterPass)
	if err != nil {
		return nil, nil, err
	}

	dek, err := crypto.DecryptAESGCM(entry.EncryptedDek, masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decryption failed (wrong master pass?): %w", err)
	}

	payload, err := crypto.DecryptAESGCM(entry.Payload, dek)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt payload: %w", err)
	}

	metaBytes, err := crypto.DecryptAESGCM(entry.Meta, dek)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt meta: %w", err)
	}

	var meta EntryMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, nil, fmt.Errorf("unmarshal meta: %w", err)
	}

	return payload, &meta, nil
}
