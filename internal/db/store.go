package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"redpacket/internal/observability/metrics"
)

// Store wraps a pgx connection pool and exposes typed helpers.
type Store struct {
	pool *pgxpool.Pool
}

// CampaignInventoryInput is used when seeding campaign inventory rows.
type CampaignInventoryInput struct {
	Amount int
	Count  int
}

// CampaignInventory is read from the campaign_inventory table.
type CampaignInventory struct {
	ID           int64
	CampaignID   int64
	Amount       int
	InitialTotal int
	OpenedCount  int
}

// ClaimLog holds data for claim_log insertions.
type ClaimLog struct {
	UserID     string
	CampaignID int64
	Amount     int
}

// New creates a Store backed by a pgx pool.
func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

// Close releases underlying connections.
func (s *Store) Close() {
	s.pool.Close()
}

// Ping verifies connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// EnsureSchema guarantees required tables exist.
func (s *Store) EnsureSchema(ctx context.Context) error {
	start := time.Now()
	defer metrics.ObserveDBOperation("ensure_schema", time.Since(start))
	_, err := s.pool.Exec(ctx, schemaSQL)
	return err
}

// RunInTx executes fn within a transaction boundary.
func (s *Store) RunInTx(ctx context.Context, fn func(pgx.Tx) error) error {
	start := time.Now()
	defer metrics.ObserveDBOperation("run_in_tx", time.Since(start))
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// InsertCampaignTx inserts a campaign row inside an existing transaction.
func (s *Store) InsertCampaignTx(ctx context.Context, tx pgx.Tx, name string, startTime, endTime time.Time) (int64, error) {
	start := time.Now()
	defer metrics.ObserveDBOperation("insert_campaign", time.Since(start))
	var id int64
	if err := tx.QueryRow(ctx, `
        INSERT INTO campaign (name, start_time, end_time, created_at)
        VALUES ($1, $2, $3, NOW())
        RETURNING id
    `, name, startTime, endTime).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// InsertCampaignInventoryTx seeds campaign inventory rows within a tx.
func (s *Store) InsertCampaignInventoryTx(ctx context.Context, tx pgx.Tx, campaignID int64, inventory []CampaignInventoryInput) error {
	start := time.Now()
	defer metrics.ObserveDBOperation("insert_campaign_inventory", time.Since(start))
	if len(inventory) == 0 {
		return errors.New("inventory required")
	}
	batch := &pgx.Batch{}
	for _, inv := range inventory {
		batch.Queue(`
            INSERT INTO campaign_inventory (campaign_id, amount, initial_total)
            VALUES ($1, $2, $3)
        `, campaignID, inv.Amount, inv.Count)
	}
	br := tx.SendBatch(ctx, batch)
	return br.Close()
}

// ListCampaignInventory fetches per-amount inventory rows.
func (s *Store) ListCampaignInventory(ctx context.Context, campaignID int64) ([]CampaignInventory, error) {
	start := time.Now()
	defer metrics.ObserveDBOperation("list_campaign_inventory", time.Since(start))
	rows, err := s.pool.Query(ctx, `
        SELECT id, campaign_id, amount, initial_total, opened_count
        FROM campaign_inventory
        WHERE campaign_id = $1
        ORDER BY amount DESC
    `, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CampaignInventory
	for rows.Next() {
		var inv CampaignInventory
		if err := rows.Scan(&inv.ID, &inv.CampaignID, &inv.Amount, &inv.InitialTotal, &inv.OpenedCount); err != nil {
			return nil, err
		}
		items = append(items, inv)
	}
	return items, rows.Err()
}

// InsertClaimLog stores a claim event for auditing.
func (s *Store) InsertClaimLog(ctx context.Context, logEntry ClaimLog) error {
	start := time.Now()
	defer metrics.ObserveDBOperation("insert_claim_log", time.Since(start))
	_, err := s.pool.Exec(ctx, `
        INSERT INTO claim_log (user_id, campaign_id, amount)
        VALUES ($1, $2, $3)
    `, logEntry.UserID, logEntry.CampaignID, logEntry.Amount)
	return err
}

// IncrementOpenedCount bumps opened_count for the claimed amount.
func (s *Store) IncrementOpenedCount(ctx context.Context, campaignID int64, amount int) error {
	start := time.Now()
	defer metrics.ObserveDBOperation("increment_opened_count", time.Since(start))
	cmdTag, err := s.pool.Exec(ctx, `
        UPDATE campaign_inventory
        SET opened_count = opened_count + 1
        WHERE campaign_id = $1 AND amount = $2
    `, campaignID, amount)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("campaign inventory row not found")
	}
	return nil
}
