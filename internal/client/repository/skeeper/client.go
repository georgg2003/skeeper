// Package skeeper implements a gRPC client for the Skeeper secrets sync service.
package skeeper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

// ErrVaultCryptoNotFound is returned by GetVaultCrypto when the server has no profile for this user.
var ErrVaultCryptoNotFound = errors.New("vault crypto not found on server")

// TokenProvider supplies a bearer access token for authenticated RPCs.
type TokenProvider interface {
	GetValidToken(ctx context.Context) (string, error)
}

// SkeeperClient performs encrypted entry sync against the Skeeper service.
type SkeeperClient struct {
	conn *grpc.ClientConn
	api  api.SkeeperClient
}

func newAuthInterceptor(provider TokenProvider) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		token, err := provider.GetValidToken(ctx)
		if err != nil {
			return err
		}

		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// Sync uploads local dirty entries and returns server-side updates since lastSyncAt.
func (c *SkeeperClient) Sync(ctx context.Context, entries []models.Entry, lastSyncAt time.Time) ([]models.Entry, error) {
	updates := make([]*api.Entry, 0, len(entries))
	for i := range entries {
		updates = append(updates, clientEntryToProto(&entries[i]))
	}

	resp, err := c.api.Sync(ctx, api.SyncRequest_builder{
		Updates:    updates,
		LastSyncAt: timestamppb.New(lastSyncAt),
	}.Build())
	if err != nil {
		return nil, err
	}

	out := make([]models.Entry, 0, len(resp.GetUpdates()))
	for _, pe := range resp.GetUpdates() {
		e, err := protoToClientEntry(pe)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func clientEntryToProto(e *models.Entry) *api.Entry {
	return api.Entry_builder{
		Uuid:         e.UUID.String(),
		Type:         e.Type,
		EncryptedDek: e.EncryptedDek,
		Payload:      e.Payload,
		Meta:         e.Meta,
		Version:      e.Version,
		IsDeleted:    e.IsDeleted,
		UpdatedAt:    timestamppb.New(e.UpdatedAt),
	}.Build()
}

func protoToClientEntry(pe *api.Entry) (models.Entry, error) {
	id, err := uuid.Parse(pe.GetUuid())
	if err != nil {
		return models.Entry{}, err
	}
	updatedAt := time.Time{}
	if pe.GetUpdatedAt() != nil {
		updatedAt = pe.GetUpdatedAt().AsTime()
	}
	return models.Entry{
		UUID:         id,
		Type:         pe.GetType(),
		EncryptedDek: pe.GetEncryptedDek(),
		Payload:      pe.GetPayload(),
		Meta:         pe.GetMeta(),
		Version:      pe.GetVersion(),
		IsDeleted:    pe.GetIsDeleted(),
		UpdatedAt:    updatedAt,
	}, nil
}

// NewSkeeperClient dials Skeeper with a token-attaching unary interceptor.
func NewSkeeperClient(addr string, provider TokenProvider) (*SkeeperClient, error) {
	if addr == "" {
		return nil, fmt.Errorf("skeeper address is required")
	}
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(newAuthInterceptor(provider)),
	)
	if err != nil {
		return nil, err
	}

	return &SkeeperClient{
		conn: conn,
		api:  api.NewSkeeperClient(conn),
	}, nil
}

// GetVaultCrypto returns the server-stored KDF salt and master-key verifier for the authenticated user.
func (c *SkeeperClient) GetVaultCrypto(ctx context.Context) (kdfSalt, masterVerifier []byte, err error) {
	resp, err := c.api.GetVaultCrypto(ctx, api.GetVaultCryptoRequest_builder{}.Build())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil, ErrVaultCryptoNotFound
		}
		return nil, nil, err
	}
	v := resp.GetVault()
	if v == nil || len(v.GetKdfSalt()) == 0 || len(v.GetMasterVerifier()) == 0 {
		return nil, nil, ErrVaultCryptoNotFound
	}
	return v.GetKdfSalt(), v.GetMasterVerifier(), nil
}

// PutVaultCrypto uploads salt and verifier; the server treats identical repeats as success.
func (c *SkeeperClient) PutVaultCrypto(ctx context.Context, kdfSalt, masterVerifier []byte) error {
	_, err := c.api.PutVaultCrypto(ctx, api.PutVaultCryptoRequest_builder{
		Vault: api.VaultCrypto_builder{
			KdfSalt:        kdfSalt,
			MasterVerifier: masterVerifier,
		}.Build(),
	}.Build())
	return err
}
