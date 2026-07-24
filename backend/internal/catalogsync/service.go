package catalogsync

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SyncService coordinates catalog upserts and event publication.
type SyncService struct {
	Nodes      *NodeRepository
	Edges      *EdgeRepository
	Publisher  EventPublisher
	SourceName string
}

func (s *SyncService) UpsertNodes(ctx context.Context, nodes []NodeInput) error {
	for _, n := range nodes {
		hash, err := ComputeNodeHash(n)
		if err != nil {
			return fmt.Errorf("hash node %s: %w", n.Name, err)
		}

		res, err := s.Nodes.Upsert(ctx, n, hash)
		if err != nil {
			return fmt.Errorf("upsert node %s: %w", n.Name, err)
		}
		if res.Change == ChangeNone {
			continue
		}

		evt := CatalogChangeEvent{
			EventID:    uuid.NewString(),
			EntityType: "catalog_node",
			ChangeType: res.Change,
			TenantID:   n.TenantID.String(),
			OccurredAt: time.Now().UTC(),
			Before:     res.Before,
			After:      res.After,
			Source:     s.SourceName,
		}
		if err := s.Publisher.Publish(ctx, evt); err != nil {
			return fmt.Errorf("publish node event %s: %w", n.Name, err)
		}
	}
	return nil
}

func (s *SyncService) UpsertEdges(ctx context.Context, edges []EdgeInput) error {
	for _, e := range edges {
		hash, err := ComputeEdgeHash(e)
		if err != nil {
			return fmt.Errorf("hash edge %s: %w", e.EdgeType, err)
		}

		res, err := s.Edges.Upsert(ctx, e, hash)
		if err != nil {
			return fmt.Errorf("upsert edge %s: %w", e.EdgeType, err)
		}
		if res.Change == ChangeNone {
			continue
		}

		evt := CatalogChangeEvent{
			EventID:    uuid.NewString(),
			EntityType: "catalog_edge",
			ChangeType: res.Change,
			TenantID:   e.TenantID.String(),
			OccurredAt: time.Now().UTC(),
			Before:     res.Before,
			After:      res.After,
			Source:     s.SourceName,
		}
		if err := s.Publisher.Publish(ctx, evt); err != nil {
			return fmt.Errorf("publish edge event %s: %w", e.EdgeType, err)
		}
	}
	return nil
}
