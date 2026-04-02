package service

import (
	"context"
	"encoding/json"

	"github.com/wecrazy/sociomile/backend/internal/model"
	"github.com/wecrazy/sociomile/backend/internal/repository"
)

func appendDomainEvent(ctx context.Context, tx *repository.Store, tenantID string, eventType string, entityType string, entityID string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	body := string(encoded)
	if err := tx.CreateActivityLog(ctx, &model.ActivityLog{
		TenantID:   tenantID,
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		Payload:    body,
	}); err != nil {
		return err
	}

	return tx.CreateOutboxEvent(ctx, &model.OutboxEvent{
		TenantID:   tenantID,
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		RoutingKey: eventType,
		Payload:    body,
	})
}
