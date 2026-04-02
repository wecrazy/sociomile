package health

import (
	"context"
	"time"

	"github.com/wecrazy/sociomile/backend/internal/cache"
)

const workerHeartbeatTTL = 20 * time.Second

// ServiceStatus describes the health state of a single runtime component.
type ServiceStatus string

const (
	// ServiceStatusOnline indicates the service is responding normally.
	ServiceStatusOnline ServiceStatus = "online"
	// ServiceStatusOffline indicates the service is known to be unavailable.
	ServiceStatusOffline ServiceStatus = "offline"
	// ServiceStatusUnknown indicates the service could not be evaluated.
	ServiceStatusUnknown ServiceStatus = "unknown"
)

// Service reports the health of a single runtime dependency.
type Service struct {
	Status    ServiceStatus `json:"status"`
	CheckedAt *time.Time    `json:"checked_at,omitempty"`
}

// Payload is the public JSON response returned by the /health endpoint.
type Payload struct {
	Status   string             `json:"status"`
	Port     int                `json:"port"`
	Services map[string]Service `json:"services"`
}

// Reporter builds runtime health payloads for the API.
type Reporter struct {
	cache *cache.Client
	now   func() time.Time
}

// NewReporter creates a health reporter backed by the shared cache client.
func NewReporter(cacheClient *cache.Client) *Reporter {
	return &Reporter{cache: cacheClient, now: time.Now}
}

// Snapshot returns the current API and worker health payload.
func (r *Reporter) Snapshot(ctx context.Context, port int) Payload {
	now := r.now().UTC()
	services := map[string]Service{
		"api": {
			Status:    ServiceStatusOnline,
			CheckedAt: timestampPointer(now),
		},
	}

	workerService := Service{Status: ServiceStatusUnknown}
	status := "ok"

	if r != nil && r.cache != nil && r.cache.Enabled() {
		workerService = r.workerSnapshot(ctx, now)
		if workerService.Status == ServiceStatusOffline {
			status = "degraded"
		}
	}

	services["worker"] = workerService

	return Payload{
		Status:   status,
		Port:     port,
		Services: services,
	}
}

// RecordWorkerHeartbeat stores the worker heartbeat in Redis so the API can surface it.
func RecordWorkerHeartbeat(ctx context.Context, cacheClient *cache.Client) error {
	if cacheClient == nil || !cacheClient.Enabled() {
		return nil
	}

	return cacheClient.SetJSON(ctx, cacheClient.Key("runtime", "worker", "heartbeat"), workerHeartbeat{
		UpdatedAt: time.Now().UTC(),
	}, workerHeartbeatTTL)
}

type workerHeartbeat struct {
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *Reporter) workerSnapshot(ctx context.Context, now time.Time) Service {
	var heartbeat workerHeartbeat
	found, err := r.cache.GetJSON(ctx, r.cache.Key("runtime", "worker", "heartbeat"), &heartbeat)
	if err != nil {
		return Service{Status: ServiceStatusUnknown}
	}

	if !found {
		return Service{Status: ServiceStatusOffline}
	}

	checkedAt := heartbeat.UpdatedAt.UTC()
	if now.Sub(heartbeat.UpdatedAt) > workerHeartbeatTTL {
		return Service{
			Status:    ServiceStatusOffline,
			CheckedAt: timestampPointer(checkedAt),
		}
	}

	return Service{
		Status:    ServiceStatusOnline,
		CheckedAt: timestampPointer(checkedAt),
	}
}

func timestampPointer(value time.Time) *time.Time {
	timestampValue := value
	return &timestampValue
}
