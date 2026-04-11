package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

func TestTokenUseCase_GetValidToken_ReturnsErrorWhenSaveFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	oldRT := "refresh-old"
	initialSess := models.Session{
		AccessToken:      "expired-access",
		RefreshToken:     oldRT,
		ExpiresAt:        time.Now().Add(-time.Hour),
		RefreshExpiresAt: time.Now().Add(time.Hour),
	}
	newSess := models.Session{
		AccessToken:      "new-access",
		RefreshToken:     "new-refresh",
		ExpiresAt:        time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
	}

	store := NewMockSessionStore(ctrl)
	auth := NewMockAuthProvider(ctrl)
	gomock.InOrder(
		store.EXPECT().GetSession(gomock.Any()).Return(&initialSess, nil),
		auth.EXPECT().Refresh(gomock.Any(), oldRT).Return(&newSess, nil),
		store.EXPECT().SaveSession(gomock.Any(), gomock.Any()).Return(errors.New("disk full")),
	)

	uc := NewTokenUseCase(store, auth, discardClientLog())
	_, err := uc.GetValidToken(ctx)
	require.Error(t, err, "expected error when SaveSession fails after refresh")
}
