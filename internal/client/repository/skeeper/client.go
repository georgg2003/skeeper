// Package skeeper is the gRPC client for vault sync and server-side vault crypto RPCs.
package skeeper

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

// ErrVaultCryptoNotFound is returned when the server has no vault row for this user yet.
var ErrVaultCryptoNotFound = errors.New("vault crypto not found on server")

type TokenProvider interface {
	GetValidToken(ctx context.Context) (string, error)
}

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

func (c *SkeeperClient) Sync(ctx context.Context, entries []models.Entry, lastSyncAt time.Time) ([]models.Entry, []uuid.UUID, error) {
	updates := make([]*api.Entry, 0, len(entries))
	for i := range entries {
		updates = append(updates, clientEntryToProto(&entries[i]))
	}

	resp, err := c.api.Sync(ctx, api.SyncRequest_builder{
		Updates:    updates,
		LastSyncAt: timestamppb.New(lastSyncAt),
	}.Build())
	if err != nil {
		return nil, nil, err
	}

	out := make([]models.Entry, 0, len(resp.GetUpdates()))
	for _, pe := range resp.GetUpdates() {
		e, err := protoToClientEntry(pe)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, e)
	}

	rawApplied := resp.GetAppliedUpdateUuids()
	applied := make([]uuid.UUID, 0, len(rawApplied))
	for _, s := range rawApplied {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, nil, fmt.Errorf("sync applied_update_uuids: %w", err)
		}
		applied = append(applied, id)
	}

	return out, applied, nil
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

// NewSkeeperClient dials addr and attaches the access token on every unary call. dialOpts must
// include transport credentials from grpcclient.DialOptions.
func NewSkeeperClient(addr string, provider TokenProvider, dialOpts ...grpc.DialOption) (*SkeeperClient, error) {
	if addr == "" {
		return nil, fmt.Errorf("skeeper address is required")
	}
	if len(dialOpts) == 0 {
		return nil, fmt.Errorf("dial options are required (use grpcclient.DialOptions)")
	}
	opts := append([]grpc.DialOption(nil), dialOpts...)
	opts = append(opts, grpc.WithUnaryInterceptor(newAuthInterceptor(provider)))
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &SkeeperClient{
		conn: conn,
		api:  api.NewSkeeperClient(conn),
	}, nil
}

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

func (c *SkeeperClient) PutVaultCrypto(ctx context.Context, kdfSalt, masterVerifier []byte) error {
	_, err := c.api.PutVaultCrypto(ctx, api.PutVaultCryptoRequest_builder{
		Vault: api.VaultCrypto_builder{
			KdfSalt:        kdfSalt,
			MasterVerifier: masterVerifier,
		}.Build(),
	}.Build())
	return err
}

func (c *SkeeperClient) Close(ctx context.Context) error {
	return c.conn.Close()
}
