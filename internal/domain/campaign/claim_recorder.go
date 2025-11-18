package campaign

import (
	"context"
	"log"
	"time"

	"redpacket/internal/db"
	"redpacket/internal/observability/metrics"
)

// ClaimRecorder persists claim events using the store.
type ClaimRecorder struct {
	store *db.Store
}

// NewClaimRecorder builds a recorder.
func NewClaimRecorder(store *db.Store) *ClaimRecorder {
	return &ClaimRecorder{store: store}
}

// HandleClaim processes a claim event by inserting logs and updating counters.
func (r *ClaimRecorder) HandleClaim(ctx context.Context, event ClaimEvent) error {
	start := time.Now()
	defer metrics.ObserveConsumerProcessing("handle_claim", time.Since(start))
	if err := r.store.InsertClaimLog(ctx, db.ClaimLog{
		UserID:     event.UserID,
		CampaignID: event.CampaignID,
		Amount:     event.Amount,
	}); err != nil {
		log.Printf("claim recorder: failed to insert log for campaign=%d user=%s: %v", event.CampaignID, event.UserID, err)
		return err
	}
	if err := r.store.IncrementOpenedCount(ctx, event.CampaignID, event.Amount); err != nil {
		log.Printf("claim recorder: failed to increment opened count for campaign=%d amount=%d: %v", event.CampaignID, event.Amount, err)
		return err
	}
	return nil
}
