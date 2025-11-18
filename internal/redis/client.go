package redis

import (
	"context"
	"fmt"
	"time"

	goRedis "github.com/redis/go-redis/v9"

	"redpacket/internal/observability/metrics"
	"redpacket/scripts/lua"
)

// Client wraps go-redis and exposes helpers for campaign keys and Lua execution.
type Client struct {
	rdb         *goRedis.Client
	claimScript *goRedis.Script
}

// New creates a Redis client and verifies connectivity.
func New(addr string) (*Client, error) {
	rdb := goRedis.NewClient(&goRedis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Client{
		rdb:         rdb,
		claimScript: goRedis.NewScript(lua.ClaimScript),
	}, nil
}

// Close shuts down the underlying Redis client.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// RunClaimScript executes the Lua script atomically.
func (c *Client) RunClaimScript(ctx context.Context, keys []string, args ...interface{}) ([]interface{}, error) {
	start := time.Now()
	defer metrics.ObserveRedisOperation("run_claim_script", time.Since(start))
	result, err := c.claimScript.Run(ctx, c.rdb, keys, args...).Result()
	if err != nil {
		return nil, err
	}
	arr, ok := result.([]interface{})
	if !ok || len(arr) != 2 {
		return nil, fmt.Errorf("unexpected Lua script response: %v", result)
	}
	return arr, nil
}

// InitializeInventory seeds Redis with campaign inventory counters and clears opened set.
func (c *Client) InitializeInventory(ctx context.Context, campaignID int64, inventory map[int]int) error {
	start := time.Now()
	defer metrics.ObserveRedisOperation("initialize_inventory", time.Since(start))
	pipe := c.rdb.TxPipeline()
	pipe.Del(ctx, c.OpenedKey(campaignID))
	pipe.Del(ctx, c.AmountsKey(campaignID))
	for amount, count := range inventory {
		pipe.Set(ctx, c.InventoryKey(campaignID, amount), count, 0)
		pipe.SAdd(ctx, c.AmountsKey(campaignID), amount)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// SetCampaignWindow stores the active window meta in Redis.
func (c *Client) SetCampaignWindow(ctx context.Context, campaignID int64, start, end time.Time) error {
	startTime := time.Now()
	defer metrics.ObserveRedisOperation("set_campaign_window", time.Since(startTime))
	key := c.CampaignWindowKey(campaignID)
	return c.rdb.HSet(ctx, key, map[string]interface{}{
		"start": start.Unix(),
		"end":   end.Unix(),
	}).Err()
}

// OpenedKey returns the Redis key that tracks which users already opened a campaign.
func (c *Client) OpenedKey(campaignID int64) string {
	return fmt.Sprintf("campaign:%d:opened", campaignID)
}

// InventoryKey returns the Redis key used to store inventory per amount.
func (c *Client) InventoryKey(campaignID int64, amount int) string {
	return fmt.Sprintf("campaign:%d:inv:%d", campaignID, amount)
}

// AmountsKey stores the configured amounts for a campaign.
func (c *Client) AmountsKey(campaignID int64) string {
	return fmt.Sprintf("campaign:%d:amounts", campaignID)
}

// CampaignWindowKey stores the start and end timestamps.
func (c *Client) CampaignWindowKey(campaignID int64) string {
	return fmt.Sprintf("campaign:%d:window", campaignID)
}
