package cache

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestCacheJSONHelpersAndVersioning(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	cacheClient := New(client)
	require.True(t, cacheClient.Enabled())
	require.Equal(t, "sociomile:tenant:t-1:conversations", cacheClient.Key("tenant", "t-1", "conversations"))

	type payload struct {
		Name string `json:"name"`
	}

	stored := payload{Name: "cached"}
	require.NoError(t, cacheClient.SetJSON(ctx, "payload", stored, time.Minute))

	var loaded payload
	hit, err := cacheClient.GetJSON(ctx, "payload", &loaded)
	require.NoError(t, err)
	require.True(t, hit)
	require.Equal(t, stored, loaded)

	var missing payload
	hit, err = cacheClient.GetJSON(ctx, "missing", &missing)
	require.NoError(t, err)
	require.False(t, hit)

	server.Set("bad-json", "{")
	_, err = cacheClient.GetJSON(ctx, "bad-json", &loaded)
	require.Error(t, err)

	require.NoError(t, New(nil).SetJSON(ctx, "ignored", stored, time.Minute))

	type invalidPayload struct {
		Channel chan int `json:"channel"`
	}
	require.Error(t, cacheClient.SetJSON(ctx, "invalid", invalidPayload{Channel: make(chan int)}, time.Minute))

	require.EqualValues(t, 1, cacheClient.Version(ctx, "tenant-a", "tickets"))
	server.Set(cacheClient.Key("tenant", "tenant-a", "tickets", "version"), "7")
	require.EqualValues(t, 7, cacheClient.Version(ctx, "tenant-a", "tickets"))

	cacheClient.BumpVersion(ctx, "tenant-a", "tickets")
	require.EqualValues(t, 8, cacheClient.Version(ctx, "tenant-a", "tickets"))

	require.EqualValues(t, 1, New(nil).Version(ctx, "tenant-b", "tickets"))
	New(nil).BumpVersion(ctx, "tenant-b", "tickets")

	require.NoError(t, client.Close())
	require.EqualValues(t, 1, cacheClient.Version(ctx, "tenant-a", "tickets"))
	_, err = cacheClient.GetJSON(ctx, "payload", &loaded)
	require.Error(t, err)
}

func TestCacheRateLimitBranches(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	cacheClient := New(client)
	allowed, err := cacheClient.CheckRateLimit(ctx, "rate-limit", 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)
	require.True(t, server.Exists("rate-limit"))

	allowed, err = cacheClient.CheckRateLimit(ctx, "rate-limit", 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = cacheClient.CheckRateLimit(ctx, "rate-limit", 2, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = New(nil).CheckRateLimit(ctx, "disabled", 1, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)

	require.NoError(t, client.Close())
	allowed, err = cacheClient.CheckRateLimit(ctx, "closed", 1, time.Minute)
	require.Error(t, err)
	require.False(t, allowed)
}
