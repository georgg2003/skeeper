package delivery

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func TestDelivery_List_empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().ListLocal(gomock.Any()).Return([]models.Entry{}, nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader(""))
	require.NoError(t, d.List(cmd, nil))
	assert.Contains(t, out.String(), "No entries.")
}

func TestDelivery_List_rows(t *testing.T) {
	id := uuid.New()
	ts := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().ListLocal(gomock.Any()).Return([]models.Entry{{
		UUID: id, Type: models.EntryTypePassword, UpdatedAt: ts, IsDirty: true,
	}}, nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader(""))
	require.NoError(t, d.List(cmd, nil))
	assert.Contains(t, out.String(), id.String())
}

func TestDelivery_List_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().ListLocal(gomock.Any()).Return(nil, context.Canceled)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, _ := testCmd(strings.NewReader(""))
	require.Error(t, d.List(cmd, nil))
}
