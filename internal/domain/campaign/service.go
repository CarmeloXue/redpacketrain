package campaign

import (
	"context"
	"errors"
	"fmt"
	"time"

	"redpacket/internal/db"
	redisClient "redpacket/internal/redis"

	"github.com/jackc/pgx/v5"
)

const (
	StatusOK               = "OK"
	StatusAlreadyOpened    = "ALREADY_OPENED"
	StatusSoldOut          = "SOLD_OUT"
	StatusCampaignInactive = "CAMPAIGN_INACTIVE"
	StatusCampaignNotFound = "CAMPAIGN_NOT_FOUND"
)

// ErrCampaignNotFound indicates the campaign is missing.
var ErrCampaignNotFound = errors.New("campaign not found")

// ErrCampaignInactive indicates the campaign is outside its schedule.
var ErrCampaignInactive = errors.New("campaign not active")

// Service coordinates DB + Redis operations.
type Service struct {
	store *db.Store
	redis *redisClient.Client
}

// CreateInput captures campaign creation payload.
type CreateInput struct {
	Name      string
	Inventory map[int]int
	StartTime time.Time
	EndTime   time.Time
}

// OpenResult represents the outcome of opening a red packet.
type OpenResult struct {
	Status string
	Amount int
}

// NewService wires dependencies.
func NewService(store *db.Store, redis *redisClient.Client) *Service {
	return &Service{store: store, redis: redis}
}

// CreateCampaign persists metadata and primes Redis inventory.
func (s *Service) CreateCampaign(ctx context.Context, in CreateInput) (int64, error) {
	if in.Name == "" {
		return 0, errors.New("name is required")
	}
	if len(in.Inventory) == 0 {
		return 0, errors.New("inventory is required")
	}
	if in.StartTime.IsZero() || in.EndTime.IsZero() {
		return 0, errors.New("start and end time required")
	}
	if !in.EndTime.After(in.StartTime) {
		return 0, errors.New("end time must be after start time")
	}

	entries := make([]db.CampaignInventoryInput, 0, len(in.Inventory))
	for amount, count := range in.Inventory {
		if amount <= 0 {
			return 0, fmt.Errorf("invalid amount %d", amount)
		}
		if count <= 0 {
			return 0, fmt.Errorf("invalid count for amount %d", amount)
		}
		entries = append(entries, db.CampaignInventoryInput{Amount: amount, Count: count})
	}

	var campaignID int64
	if err := s.store.RunInTx(ctx, func(tx pgx.Tx) error {
		id, err := s.store.InsertCampaignTx(ctx, tx, in.Name, in.StartTime, in.EndTime)
		if err != nil {
			return err
		}
		if err := s.store.InsertCampaignInventoryTx(ctx, tx, id, entries); err != nil {
			return err
		}
		campaignID = id
		return nil
	}); err != nil {
		return 0, err
	}
	if err := s.redis.InitializeInventory(ctx, campaignID, in.Inventory); err != nil {
		return 0, err
	}
	if err := s.redis.SetCampaignWindow(ctx, campaignID, in.StartTime, in.EndTime); err != nil {
		return 0, err
	}
	return campaignID, nil
}

// OpenRedPacket runs the Lua script to atomically assign an amount.
func (s *Service) OpenRedPacket(ctx context.Context, campaignID int64, userID string) (*OpenResult, error) {
	if userID == "" {
		return nil, errors.New("user id required")
	}
	keys := []string{
		s.redis.OpenedKey(campaignID),
		s.redis.CampaignWindowKey(campaignID),
		s.redis.AmountsKey(campaignID),
	}
	args := []interface{}{userID, time.Now().Unix(), fmt.Sprintf("%d", campaignID)}

	resp, err := s.redis.RunClaimScript(ctx, keys, args...)
	if err != nil {
		return nil, err
	}

	status := fmt.Sprintf("%v", resp[0])
	amount := parseAmount(resp[1])

	switch status {
	case StatusCampaignNotFound:
		return nil, ErrCampaignNotFound
	case StatusCampaignInactive:
		return nil, ErrCampaignInactive
	}

	return &OpenResult{Status: status, Amount: amount}, nil
}

func parseAmount(value interface{}) int {
	switch v := value.(type) {
	case int64:
		return int(v)
	case int32:
		return int(v)
	case float64:
		return int(v)
	case string:
		var amt int
		fmt.Sscanf(v, "%d", &amt)
		return amt
	default:
		return 0
	}
}
