package campaign

import "time"

// ClaimEvent encapsulates the data emitted after a user claim succeeds.
type ClaimEvent struct {
	UserID     string    `json:"user_id"`
	CampaignID int64     `json:"campaign_id"`
	Amount     int       `json:"amount"`
	Timestamp  time.Time `json:"ts"`
}
