package router

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"redpacket/internal/domain/campaign"
	"redpacket/internal/messaging/claim"
	"redpacket/internal/observability/metrics"
)

// Dependencies enumerates services required by API handlers.
type Dependencies struct {
	CampaignService *campaign.Service
	Publisher       *claim.Publisher
}

// New builds a gin.Engine with all routes registered.
func New(deps Dependencies) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), metrics.GinMiddleware())
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	h := &handler{svc: deps.CampaignService, publisher: deps.Publisher}

	router.POST("/campaign", h.createCampaign)
	router.POST("/campaign/:id/open", h.openRedPacket)

	return router
}

type handler struct {
	svc       *campaign.Service
	publisher *claim.Publisher
}

type createCampaignRequest struct {
	Name      string         `json:"name" binding:"required"`
	Inventory map[string]int `json:"inventory" binding:"required"`
	StartTime time.Time      `json:"start_time" binding:"required"`
	EndTime   time.Time      `json:"end_time" binding:"required"`
}

type createCampaignResponse struct {
	ID int64 `json:"id"`
}

type openRedPacketRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *handler) createCampaign(c *gin.Context) {
	var req createCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	inventory := make(map[int]int, len(req.Inventory))
	for amountStr, count := range req.Inventory {
		amount, err := strconv.Atoi(amountStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "inventory keys must be integers"})
			return
		}
		inventory[amount] = count
	}
	id, err := h.svc.CreateCampaign(c.Request.Context(), campaign.CreateInput{
		Name:      req.Name,
		Inventory: inventory,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, createCampaignResponse{ID: id})
}

func (h *handler) openRedPacket(c *gin.Context) {
	campaignID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign id"})
		return
	}
	var req openRedPacketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.OpenRedPacket(c.Request.Context(), campaignID, req.UserID)
	if err != nil {
		if errors.Is(err, campaign.ErrCampaignNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, campaign.ErrCampaignInactive) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	switch result.Status {
	case campaign.StatusAlreadyOpened:
		c.JSON(http.StatusConflict, gin.H{"status": result.Status})
		return
	case campaign.StatusSoldOut:
		c.JSON(http.StatusGone, gin.H{"status": result.Status})
		return
	case campaign.StatusOK:
		event := campaign.ClaimEvent{
			UserID:     req.UserID,
			CampaignID: campaignID,
			Amount:     result.Amount,
			Timestamp:  time.Now().UTC(),
		}
		if err := h.publisher.Publish(c.Request.Context(), event); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue claim"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": result.Status, "amount": result.Amount})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": result.Status})
	}
}
