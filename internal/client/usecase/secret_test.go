package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSecretUseCase_FileTooLarge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	local := NewMockLocalSecretStore(ctrl)
	sess := NewMockSessionReader(ctrl)
	uc := NewSecretUseCase(local, sess, nil, discardClientLog(), 4)
	ctx := context.Background()
	err := uc.SetFile(ctx, EntryMetadata{Name: "x"}, "a.bin", []byte("12345"), "m")
	require.Error(t, err, "expected error")
}
