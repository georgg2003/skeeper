package cli

//go:generate go tool mockgen -typed -destination=mock_commands_test.go -package=cli -source=usecase_ports.go AuthCommands,SecretCommands,SyncCommands

import (
	"context"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/google/uuid"
)

// Narrow interfaces keep Cobra commands testable without a live vault or network.

type AuthCommands interface {
	Register(ctx context.Context, email, password string) error
	Login(ctx context.Context, login, password string) error
	Logout(ctx context.Context) error
	GetValidToken(ctx context.Context) (string, error)
}

type SecretCommands interface {
	ListLocal(ctx context.Context) ([]models.Entry, error)
	GetLocalEntry(ctx context.Context, id uuid.UUID) (models.Entry, error)
	GetDecryptedEntry(ctx context.Context, id uuid.UUID, masterPass string) ([]byte, *usecase.EntryMetadata, error)
	SetPassword(ctx context.Context, meta usecase.EntryMetadata, password, masterPass string) error
	SetText(ctx context.Context, meta usecase.EntryMetadata, text, masterPass string) error
	SetBinary(ctx context.Context, meta usecase.EntryMetadata, data []byte, masterPass string) error
	SetCard(ctx context.Context, meta usecase.EntryMetadata, card models.CardPayload, masterPass string) error
}

type SyncCommands interface {
	Sync(ctx context.Context) error
}
