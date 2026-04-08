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

var ErrWrongMasterPassword = errors.New("wrong master password")

type EntryMetadata struct {
	Name      string            `json:"name"`
	Notes     string            `json:"notes,omitempty"`
	ExtraTags map[string]string `json:"tags,omitempty"`
}

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

type SessionReader interface {
	GetSession(ctx context.Context) (*models.Session, error)
}

type VaultRemote interface {
	GetVaultCrypto(ctx context.Context) (kdfSalt, masterVerifier []byte, err error)
	PutVaultCrypto(ctx context.Context, kdfSalt, masterVerifier []byte) error
}

type SecretUseCase struct {
	local    LocalSecretStore
	sessions SessionReader
	remote   VaultRemote
	log      *slog.Logger
}

// NewSecretUseCase remote can be nil if you never sync vault crypto to the server.
func NewSecretUseCase(local LocalSecretStore, sessions SessionReader, remote VaultRemote, log *slog.Logger) *SecretUseCase {
	return &SecretUseCase{
		local:    local,
		sessions: sessions,
		remote:   remote,
		log:      log.With("component", "secret_usecase"),
	}
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

func (uc *SecretUseCase) SetPassword(ctx context.Context, meta EntryMetadata, password string, masterPass string) error {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return err
	}
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

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("new entry id: %w", err)
	}
	u := uid
	entry := models.Entry{
		UUID:         id,
		Type:         models.EntryTypePassword,
		Payload:      encryptedPayload,
		EncryptedDek: encryptedDEK,
		Meta:         encryptedMeta,
		Version:      1,
		UpdatedAt:    time.Now(),
		IsDeleted:    false,
		UserID:       &u,
	}

	if err := uc.local.SaveEntry(ctx, entry, true); err != nil {
		return err
	}
	return uc.finalizeVaultVerifier(ctx, uid, salt, storedVerifier, masterKey)
}

func (uc *SecretUseCase) SetText(ctx context.Context, meta EntryMetadata, text string, masterPass string) error {
	return uc.setBlob(ctx, models.EntryTypeText, meta, []byte(text), masterPass)
}

func (uc *SecretUseCase) SetBinary(ctx context.Context, meta EntryMetadata, data []byte, masterPass string) error {
	return uc.setBlob(ctx, models.EntryTypeBinary, meta, data, masterPass)
}

func (uc *SecretUseCase) SetCard(ctx context.Context, meta EntryMetadata, card models.CardPayload, masterPass string) error {
	payload, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}
	return uc.setBlob(ctx, models.EntryTypeCard, meta, payload, masterPass)
}

func (uc *SecretUseCase) setBlob(ctx context.Context, typ string, meta EntryMetadata, plaintext []byte, masterPass string) error {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return err
	}
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

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("new entry id: %w", err)
	}
	u := uid
	entry := models.Entry{
		UUID:         id,
		Type:         typ,
		Payload:      encryptedPayload,
		EncryptedDek: encryptedDEK,
		Meta:         encryptedMeta,
		Version:      1,
		UpdatedAt:    time.Now(),
		IsDeleted:    false,
		UserID:       &u,
	}

	if err := uc.local.SaveEntry(ctx, entry, true); err != nil {
		return err
	}
	return uc.finalizeVaultVerifier(ctx, uid, salt, storedVerifier, masterKey)
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

func (uc *SecretUseCase) ListLocal(ctx context.Context) ([]models.Entry, error) {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.local.ListEntries(ctx, &uid)
}

func (uc *SecretUseCase) GetLocalEntry(ctx context.Context, id uuid.UUID) (models.Entry, error) {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return models.Entry{}, err
	}
	return uc.local.GetEntry(ctx, id, &uid)
}

func (uc *SecretUseCase) GetDecryptedEntry(ctx context.Context, id uuid.UUID, masterPass string) ([]byte, *EntryMetadata, error) {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return nil, nil, err
	}
	entry, err := uc.local.GetEntry(ctx, id, &uid)
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
