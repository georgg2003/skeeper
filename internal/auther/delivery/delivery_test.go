package delivery

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	usecase "github.com/georgg2003/skeeper/internal/auther/usecase"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestAutherServer_CreateUser(t *testing.T) {
	emailOK, passOK := "ok@example.com", "okpassword1234"
	valErr := errors.NewValidationError("email", "invalid")

	tests := []struct {
		name     string
		setup    func(m *MockUseCase)
		email    string
		password string
		wantCode codes.Code
		wantOK   bool
		wantID   int64
	}{
		{
			name: "validation_error_invalid_argument",
			setup: func(m *MockUseCase) {
				m.EXPECT().CreateUser(gomock.Any(), models.UserCredentials{Email: "bad", Password: "x"}).
					Return(models.UserInfo{}, valErr)
			},
			email:    "bad",
			password: "x",
			wantCode: codes.InvalidArgument,
		},
		{
			name: "already_exists",
			setup: func(m *MockUseCase) {
				m.EXPECT().CreateUser(gomock.Any(), models.UserCredentials{Email: emailOK, Password: passOK}).
					Return(models.UserInfo{}, usecase.ErrUserExists)
			},
			email:    emailOK,
			password: passOK,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "internal_on_generic_error",
			setup: func(m *MockUseCase) {
				m.EXPECT().CreateUser(gomock.Any(), models.UserCredentials{Email: emailOK, Password: passOK}).
					Return(models.UserInfo{}, errors.New("db"))
			},
			email:    emailOK,
			password: passOK,
			wantCode: codes.Internal,
		},
		{
			name: "success",
			setup: func(m *MockUseCase) {
				m.EXPECT().CreateUser(gomock.Any(), models.UserCredentials{Email: emailOK, Password: passOK}).
					Return(models.UserInfo{ID: 42, Email: emailOK}, nil)
			},
			email:    emailOK,
			password: passOK,
			wantOK:   true,
			wantID:   42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUC := NewMockUseCase(ctrl)
			tt.setup(mockUC)
			srv := New(testLogger(), mockUC)

			resp, err := srv.CreateUser(context.Background(), api.CreateUserRequest_builder{
				Email: &tt.email, Password: &tt.password,
			}.Build())
			if tt.wantOK {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, resp.GetUser().GetId())
				assert.Equal(t, emailOK, resp.GetUser().GetEmail())
				return
			}
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestAutherServer_Login(t *testing.T) {
	email, pass := "u@x.y", "pw"

	tests := []struct {
		name     string
		loginEm  string
		loginPw  string
		setup    func(m *MockUseCase)
		wantCode codes.Code
		wantOK   bool
	}{
		{
			name:    "invalid_argument_on_validation",
			loginEm: "bad",
			loginPw: "",
			setup: func(m *MockUseCase) {
				m.EXPECT().LoginUser(gomock.Any(), models.UserCredentials{Email: "bad", Password: ""}).
					Return(models.LoginResponse{}, errors.NewValidationError("password", "empty"))
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "unauthenticated_invalid_credentials",
			setup: func(m *MockUseCase) {
				m.EXPECT().LoginUser(gomock.Any(), models.UserCredentials{Email: email, Password: pass}).
					Return(models.LoginResponse{}, usecase.ErrUserNotExist)
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "internal_on_usecase_error",
			setup: func(m *MockUseCase) {
				m.EXPECT().LoginUser(gomock.Any(), models.UserCredentials{Email: email, Password: pass}).
					Return(models.LoginResponse{}, errors.New("any"))
			},
			wantCode: codes.Internal,
		},
		{
			name: "success",
			setup: func(m *MockUseCase) {
				at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
				rt := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
				m.EXPECT().LoginUser(gomock.Any(), models.UserCredentials{Email: email, Password: pass}).
					Return(models.LoginResponse{
						User: models.UserInfo{ID: 9, Email: email},
						TokenPair: jwthelper.TokenPair{
							AccessToken:  jwthelper.Token{Token: "access-jwt", ExpiresAt: at},
							RefreshToken: jwthelper.Token{Token: "refresh-opaque", ExpiresAt: rt},
						},
					}, nil)
			},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUC := NewMockUseCase(ctrl)
			tt.setup(mockUC)
			srv := New(testLogger(), mockUC)
			em, pw := email, pass
			if tt.loginEm != "" || tt.loginPw != "" {
				em, pw = tt.loginEm, tt.loginPw
			}
			out, err := srv.Login(context.Background(), api.LoginRequest_builder{Email: &em, Password: &pw}.Build())
			if tt.wantOK {
				require.NoError(t, err)
				assert.Equal(t, int64(9), out.GetUser().GetId())
				assert.Equal(t, "access-jwt", out.GetAccessToken().GetToken())
				return
			}
			st, _ := status.FromError(err)
			assert.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestAutherServer_ExchangeToken(t *testing.T) {
	at := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	rt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)

	tests := []struct {
		name     string
		setup    func(m *MockUseCase)
		in       string
		wantCode codes.Code
		wantOK   bool
	}{
		{
			name: "invalid_refresh_unauthenticated",
			setup: func(m *MockUseCase) {
				m.EXPECT().RotateToken(gomock.Any(), "bad").Return(jwthelper.TokenPair{}, usecase.ErrInvalidToken)
			},
			in:       "bad",
			wantCode: codes.Unauthenticated,
		},
		{
			name: "internal_on_other_error",
			setup: func(m *MockUseCase) {
				m.EXPECT().RotateToken(gomock.Any(), "x").Return(jwthelper.TokenPair{}, errors.New("rotate failed"))
			},
			in:       "x",
			wantCode: codes.Internal,
		},
		{
			name: "success",
			setup: func(m *MockUseCase) {
				m.EXPECT().RotateToken(gomock.Any(), "old-refresh").
					Return(jwthelper.TokenPair{
						AccessToken:  jwthelper.Token{Token: "new-access", ExpiresAt: at},
						RefreshToken: jwthelper.Token{Token: "new-refresh", ExpiresAt: rt},
					}, nil)
			},
			in:     "old-refresh",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUC := NewMockUseCase(ctrl)
			tt.setup(mockUC)
			srv := New(testLogger(), mockUC)
			tok := tt.in
			out, err := srv.ExchangeToken(context.Background(), api.ExchangeTokenRequest_builder{RefreshToken: &tok}.Build())
			if tt.wantOK {
				require.NoError(t, err)
				assert.Equal(t, "new-access", out.GetAccessToken().GetToken())
				return
			}
			st, _ := status.FromError(err)
			assert.Equal(t, tt.wantCode, st.Code())
		})
	}
}
