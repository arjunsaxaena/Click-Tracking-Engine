package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sirupsen/logrus"

	db "project/migrations/sqlc"
)

type ClickService interface {
	HandleClick(ctx context.Context, req TrackInput) (TrackOutput, error)
}

type TrackInput struct {
	LinkID    string
	UserID    string
	GAID      string
	IDFA      string
	IP        string
	UserAgent string
	Referrer  string
}

type TrackOutput struct {
	StatusCode int
	Body       string
	RedirectURL string
}

type clickService struct {
	campaigns    *db.Queries
	fraudChecker *FraudChecker
	logger       *logrus.Logger
}

func NewClickService(c *db.Queries) ClickService {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	return &clickService{
		campaigns:    c,
		fraudChecker: NewFraudChecker(c),
		logger:       logger,
	}
}

func (s *clickService) HandleClick(ctx context.Context, req TrackInput) (TrackOutput, error) {
	if req.UserID == "" {
		return TrackOutput{StatusCode: 400, Body: "values missing"}, nil
	}

	linkID, err := uuid.Parse(req.LinkID)
	if err != nil {
		return TrackOutput{StatusCode: 200, Body: "<html><body>campaign not available</body></html>"}, nil
	}

	campaign, err := s.campaigns.GetCampaignByLinkID(ctx, linkID)
	if err != nil {
		return TrackOutput{StatusCode: 200, Body: "<html><body>campaign not available</body></html>"}, nil
	}

	if campaign.Status != db.CampaignStatusActive {
		return TrackOutput{StatusCode: 200, Body: "<html><body>campaign not available</body></html>"}, nil
	}

	now := time.Now()
	var startTime, endTime time.Time

	if campaign.StartDate.Valid {
		startTime = campaign.StartDate.Time
	} else {
		return TrackOutput{StatusCode: 200, Body: "<html><body>campaign not available</body></html>"}, nil
	}

	if campaign.EndDate.Valid {
		endTime = campaign.EndDate.Time
	} else {
		return TrackOutput{StatusCode: 200, Body: "<html><body>campaign not available</body></html>"}, nil
	}

	if now.Before(startTime) || now.After(endTime) {
		return TrackOutput{StatusCode: 200, Body: "<html><body>campaign not available</body></html>"}, nil
	}

	clickID := uuid.New()
	clickIDStr := clickID.String()

	fraudResults, blockCount := s.fraudChecker.RunChecks(ctx, req, clickIDStr)

	failedReasons := make([]string, 0)
	for _, result := range fraudResults {
		if result.Block {
			failedReasons = append(failedReasons, result.Reason)
		}
	}

	clickStatus := db.ClickStatusAllowed
	if blockCount >= 2 {
		clickStatus = db.ClickStatusFraud
	}

	substitutedURL, missingMacros := s.substituteMacros(campaign.TargetUrl, req, clickIDStr)

	if len(missingMacros) > 0 {
		clickStatus = db.ClickStatusFraud
		failedReasons = append(failedReasons, fmt.Sprintf("missing required macros: %s", strings.Join(missingMacros, ", ")))
	}

	go s.insertClickAsync(clickID, linkID, campaign.CampaignID, req, clickStatus, failedReasons)

	if clickStatus == db.ClickStatusFraud {
		return TrackOutput{
			StatusCode: 200,
			Body:       "<html><body>campaign not available</body></html>",
		}, nil
	}

	return TrackOutput{
		StatusCode:  302,
		RedirectURL: substitutedURL,
		Body:        fmt.Sprintf(`<html><head><meta http-equiv="refresh" content="0;url=%s"></head><body>Redirecting...</body></html>`, substitutedURL),
	}, nil
}

func (s *clickService) substituteMacros(targetURL string, input TrackInput, clickID string) (string, []string) {
	missingMacros := make([]string, 0)
	result := targetURL

	if strings.Contains(result, "{user_id}") {
		if input.UserID == "" {
			missingMacros = append(missingMacros, "user_id")
		} else {
			result = strings.ReplaceAll(result, "{user_id}", input.UserID)
		}
	}

	if strings.Contains(result, "{gaid}") {
		if input.GAID == "" {
			missingMacros = append(missingMacros, "gaid")
		} else {
			result = strings.ReplaceAll(result, "{gaid}", input.GAID)
		}
	}

	if strings.Contains(result, "{click_id}") {
		result = strings.ReplaceAll(result, "{click_id}", clickID)
	}

	return result, missingMacros
}

func (s *clickService) insertClickAsync(clickID uuid.UUID, linkID uuid.UUID, campaignID uuid.UUID, input TrackInput, status db.ClickStatus, fraudReasons []string) {
	ctx := context.Background()
	
	params := db.InsertClickParams{
		ClickID:          clickID,
		LinkID:           linkID,
		CampaignID:       campaignID,
		UserID:           input.UserID,
		Status:           status,
		FraudCheckFailed: fraudReasons,
	}

	if input.IP != "" {
		params.IpAddress = pgtype.Text{String: input.IP, Valid: true}
	}
	if input.UserAgent != "" {
		params.UserAgent = pgtype.Text{String: input.UserAgent, Valid: true}
	}
	if input.Referrer != "" {
		params.Referrer = pgtype.Text{String: input.Referrer, Valid: true}
	}
	if input.GAID != "" {
		params.Gaid = pgtype.Text{String: input.GAID, Valid: true}
	}
	if input.IDFA != "" {
		params.Idfa = pgtype.Text{String: input.IDFA, Valid: true}
	}

	err := s.campaigns.InsertClick(ctx, params)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"click_id":    clickID.String(),
			"link_id":     linkID.String(),
			"campaign_id": campaignID.String(),
			"user_id":     input.UserID,
			"error":       err.Error(),
		}).Error("Failed to insert click into database")
	}
}