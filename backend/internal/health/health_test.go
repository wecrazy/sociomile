package health

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/cache"
)

func TestSnapshotWithoutCacheKeepsWorkerUnknown(t *testing.T) {
	reporter := NewReporter(nil)
	payload := reporter.Snapshot(context.Background(), 8080)

	require.Equal(t, "ok", payload.Status)
	require.Equal(t, ServiceStatusOnline, payload.Services["api"].Status)
	require.Equal(t, ServiceStatusUnknown, payload.Services["worker"].Status)
}

func TestSnapshotMarksWorkerOfflineWithoutHeartbeat(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	reporter := NewReporter(cache.New(client))

	payload := reporter.Snapshot(context.Background(), 8080)

	require.Equal(t, "degraded", payload.Status)
	require.Equal(t, ServiceStatusOffline, payload.Services["worker"].Status)
	require.Nil(t, payload.Services["worker"].CheckedAt)
}

func TestRecordWorkerHeartbeatMakesWorkerOnline(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	cacheClient := cache.New(client)
	reporter := NewReporter(cacheClient)

	require.NoError(t, RecordWorkerHeartbeat(context.Background(), cacheClient))
	payload := reporter.Snapshot(context.Background(), 8080)

	require.Equal(t, "ok", payload.Status)
	require.Equal(t, ServiceStatusOnline, payload.Services["worker"].Status)
	require.NotNil(t, payload.Services["worker"].CheckedAt)
}

func TestSnapshotMarksStaleHeartbeatOffline(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	cacheClient := cache.New(client)
	reporter := NewReporter(cacheClient)

	require.NoError(t, RecordWorkerHeartbeat(context.Background(), cacheClient))
	reporter.now = func() time.Time {
		return time.Now().UTC().Add(workerHeartbeatTTL + time.Second)
	}

	payload := reporter.Snapshot(context.Background(), 8080)

	require.Equal(t, "degraded", payload.Status)
	require.Equal(t, ServiceStatusOffline, payload.Services["worker"].Status)
	require.NotNil(t, payload.Services["worker"].CheckedAt)
}
