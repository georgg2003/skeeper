// Package skeeper implements a gRPC client for the Skeeper secrets sync service.
package skeeper

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

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
