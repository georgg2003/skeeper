package delivery

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	usecase "github.com/georgg2003/skeeper/internal/auther/usecase"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCreateUser_ValidationErrorMapsToInvalidArgument(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	mockUC.EXPECT().
		CreateUser(gomock.Any(), models.UserCredentials{Email: "x@y.z", Password: "secret"}).
		Return(models.UserInfo{}, errors.NewValidationError("email", "bad"))

	srv := New(testLogger(), mockUC)
	email := "x@y.z"
	pass := "secret"
	_, err := srv.CreateUser(context.Background(), api.CreateUserRequest_builder{
		Email: &email, Password: &pass,
	}.Build())
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
}

func TestCreateUser_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	mockUC.EXPECT().
		CreateUser(gomock.Any(), models.UserCredentials{Email: "x@y.z", Password: "secret"}).
		Return(models.UserInfo{}, errors.New("boom"))

	srv := New(testLogger(), mockUC)
	email := "x@y.z"
	pass := "secret"
	_, err := srv.CreateUser(context.Background(), api.CreateUserRequest_builder{
		Email: &email, Password: &pass,
	}.Build())
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestCreateUser_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	mockUC.EXPECT().
		CreateUser(gomock.Any(), models.UserCredentials{Email: "a@b.c", Password: "p"}).
		Return(models.UserInfo{ID: 7, Email: "a@b.c"}, nil)

	srv := New(testLogger(), mockUC)
	email := "a@b.c"
	pass := "p"
	resp, err := srv.CreateUser(context.Background(), api.CreateUserRequest_builder{
		Email: &email, Password: &pass,
	}.Build())
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetUser().GetId() != 7 || resp.GetUser().GetEmail() != "a@b.c" {
		t.Fatalf("user %+v", resp.GetUser())
	}
}

func TestLogin_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	mockUC.EXPECT().
		LoginUser(gomock.Any(), models.UserCredentials{Email: "a@b.c", Password: "p"}).
		Return(models.LoginReponse{}, errors.New("db down"))

	srv := New(testLogger(), mockUC)
	email := "a@b.c"
	pass := "p"
	_, err := srv.Login(context.Background(), api.LoginRequest_builder{Email: &email, Password: &pass}.Build())
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	rt := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	mockUC.EXPECT().
		LoginUser(gomock.Any(), models.UserCredentials{Email: "u@x.y", Password: "pw"}).
		Return(models.LoginReponse{
			User: models.UserInfo{ID: 9, Email: "u@x.y"},
			TokenPair: jwthelper.TokenPair{
				AccessToken:  jwthelper.Token{Token: "access-jwt", ExpiresAt: at},
				RefreshToken: jwthelper.Token{Token: "refresh-opaque", ExpiresAt: rt},
			},
		}, nil)

	srv := New(testLogger(), mockUC)
	email := "u@x.y"
	pass := "pw"
	out, err := srv.Login(context.Background(), api.LoginRequest_builder{Email: &email, Password: &pass}.Build())
	if err != nil {
		t.Fatal(err)
	}
	if out.GetUser().GetId() != 9 {
		t.Fatal(out.GetUser())
	}
	if out.GetAccessToken().GetToken() != "access-jwt" || !out.GetAccessToken().GetExpiresAt().AsTime().Equal(at) {
		t.Fatal("access token mismatch")
	}
	if out.GetRefreshToken().GetToken() != "refresh-opaque" || !out.GetRefreshToken().GetExpiresAt().AsTime().Equal(rt) {
		t.Fatal("refresh token mismatch")
	}
}

func TestExchangeToken_InvalidRefreshMapsToUnauthenticated(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	mockUC.EXPECT().
		RotateToken(gomock.Any(), "any").
		Return(jwthelper.TokenPair{}, usecase.ErrInvalidToken)

	srv := New(testLogger(), mockUC)
	rt := "any"
	_, err := srv.ExchangeToken(context.Background(), api.ExchangeTokenRequest_builder{RefreshToken: &rt}.Build())
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("got %v", err)
	}
}

func TestExchangeToken_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	mockUC.EXPECT().
		RotateToken(gomock.Any(), "x").
		Return(jwthelper.TokenPair{}, errors.New("rotate failed"))

	srv := New(testLogger(), mockUC)
	rt := "x"
	_, err := srv.ExchangeToken(context.Background(), api.ExchangeTokenRequest_builder{RefreshToken: &rt}.Build())
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestExchangeToken_Success(t *testing.T) {
	at := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	rt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	mockUC.EXPECT().
		RotateToken(gomock.Any(), "old-refresh").
		Return(jwthelper.TokenPair{
			AccessToken:  jwthelper.Token{Token: "new-access", ExpiresAt: at},
			RefreshToken: jwthelper.Token{Token: "new-refresh", ExpiresAt: rt},
		}, nil)

	srv := New(testLogger(), mockUC)
	old := "old-refresh"
	out, err := srv.ExchangeToken(context.Background(), api.ExchangeTokenRequest_builder{RefreshToken: &old}.Build())
	if err != nil {
		t.Fatal(err)
	}
	if out.GetAccessToken().GetToken() != "new-access" {
		t.Fatal(out.GetAccessToken())
	}
}
