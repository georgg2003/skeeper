package delivery

//go:generate go tool mockgen -typed -destination=mock_ports_test.go -package=delivery -source=ports.go AuthCommands,SecretCommands,SyncCommands

import (
	"context"

	"github.com/google/uuid"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/internal/client/usecase"
)

// AuthCommands is the subset of auth use-case operations used by the delivery layer.
type AuthCommands interface {
	Register(ctx context.Context, email, password string) error
	Login(ctx context.Context, login, password string) error
	Logout(ctx context.Context) error
	GetValidToken(ctx context.Context) (string, error)
}

// SecretCommands is the vault CRUD surface used by delivery handlers.
type SecretCommands interface {
	ListLocal(ctx context.Context) ([]models.Entry, error)
	GetLocalEntry(ctx context.Context, id uuid.UUID) (models.Entry, error)
	GetDecryptedEntry(ctx context.Context, id uuid.UUID, masterPass string) ([]byte, *usecase.EntryMetadata, string, error)
	SetPassword(ctx context.Context, meta usecase.EntryMetadata, password, masterPass string) error
	SetText(ctx context.Context, meta usecase.EntryMetadata, text, masterPass string) error
	SetFile(ctx context.Context, meta usecase.EntryMetadata, originalFilename string, data []byte, masterPass string) error
	SetCard(ctx context.Context, meta usecase.EntryMetadata, card models.CardPayload, masterPass string) error
	UpdatePassword(ctx context.Context, id uuid.UUID, meta usecase.EntryMetadata, password, masterPass string) error
	UpdateText(ctx context.Context, id uuid.UUID, meta usecase.EntryMetadata, text, masterPass string) error
	UpdateCard(ctx context.Context, id uuid.UUID, meta usecase.EntryMetadata, card models.CardPayload, masterPass string) error
	UpdateFile(ctx context.Context, id uuid.UUID, meta usecase.EntryMetadata, masterPass string, replacePayload bool, newFile []byte, newOrigName string) error
	DeleteEntry(ctx context.Context, id uuid.UUID, masterPass string) error
}

// SyncCommands pushes and pulls ciphertext with the Skeeper service.
type SyncCommands interface {
	Sync(ctx context.Context) error
}
