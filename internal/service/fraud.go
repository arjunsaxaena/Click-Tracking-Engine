package service

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	db "project/migrations/sqlc"
)

type FraudCheckResult struct {
	Block  bool
	Reason string
}

type FraudCheck interface {
	Check(ctx context.Context, input TrackInput, clickID string) FraudCheckResult
	Name() string
}

type FraudChecker struct {
	checks []FraudCheck
}

func NewFraudChecker(queries *db.Queries) *FraudChecker {
	return &FraudChecker{
		checks: []FraudCheck{
			NewIPRateLimitCheck(queries),
			NewUABlocklistCheck(queries),
			NewDeviceIDBlocklistCheck(queries),
		},
	}
}

func (fc *FraudChecker) RunChecks(ctx context.Context, input TrackInput, clickID string) ([]FraudCheckResult, int) {
	results := make([]FraudCheckResult, 0, len(fc.checks))
	blockCount := 0

	for _, check := range fc.checks {
		result := check.Check(ctx, input, clickID)
		results = append(results, result)
		if result.Block {
			blockCount++
		}
	}

	return results, blockCount
}

type IPRateLimitCheck struct {
	queries *db.Queries
}

func NewIPRateLimitCheck(queries *db.Queries) *IPRateLimitCheck {
	return &IPRateLimitCheck{queries: queries}
}

func (c *IPRateLimitCheck) Name() string {
	return "ip_rate_limit"
}

func (c *IPRateLimitCheck) Check(ctx context.Context, input TrackInput, clickID string) FraudCheckResult {
	if input.IP == "" {
		return FraudCheckResult{
			Block:  false,
			Reason: "ip_address not provided",
		}
	}

	ipAddress := pgtype.Text{String: input.IP, Valid: true}
	count, err := c.queries.CountClicksByIPInLast60Seconds(ctx, ipAddress)
	if err != nil {
		return FraudCheckResult{
			Block:  false,
			Reason: "error checking IP rate limit",
		}
	}

	if count >= 100 {
		return FraudCheckResult{
			Block:  true,
			Reason: "ip_rate_limit: 100+ clicks from same IP in last 60 seconds",
		}
	}

	return FraudCheckResult{
		Block:  false,
		Reason: "ip_rate_limit: within rate limit",
	}
}

type UABlocklistCheck struct {
	queries *db.Queries
}

func NewUABlocklistCheck(queries *db.Queries) *UABlocklistCheck {
	return &UABlocklistCheck{queries: queries}
}

func (c *UABlocklistCheck) Name() string {
	return "ua_blocklist"
}

func (c *UABlocklistCheck) Check(ctx context.Context, input TrackInput, clickID string) FraudCheckResult {
	if input.UserAgent == "" {
		return FraudCheckResult{
			Block:  false,
			Reason: "user_agent not provided",
		}
	}

	userAgent := strings.ToLower(input.UserAgent)
	blockedPatterns := []string{
		"curl/",
		"wget/",
		"python-requests",
	}

	for _, pattern := range blockedPatterns {
		if strings.Contains(userAgent, pattern) {
			return FraudCheckResult{
				Block:  true,
				Reason: "ua_blocklist: user-agent matches blocked pattern",
			}
		}
	}

	return FraudCheckResult{
		Block:  false,
		Reason: "ua_blocklist: user-agent not blocked",
	}
}

type DeviceIDBlocklistCheck struct {
	queries *db.Queries
}

func NewDeviceIDBlocklistCheck(queries *db.Queries) *DeviceIDBlocklistCheck {
	return &DeviceIDBlocklistCheck{queries: queries}
}

func (c *DeviceIDBlocklistCheck) Name() string {
	return "device_id_blocklist"
}

func (c *DeviceIDBlocklistCheck) Check(ctx context.Context, input TrackInput, clickID string) FraudCheckResult {
	if input.GAID != "" {
		blocked, err := c.queries.IsBlocked(ctx, input.GAID)
		if err != nil {
			return FraudCheckResult{
				Block:  false,
				Reason: "error checking gaid blocklist",
			}
		}
		if blocked {
			return FraudCheckResult{
				Block:  true,
				Reason: "device_id_blocklist: gaid is in blocklist",
			}
		}
	}

	if input.IDFA != "" {
		blocked, err := c.queries.IsBlocked(ctx, input.IDFA)
		if err != nil {
			return FraudCheckResult{
				Block:  false,
				Reason: "error checking idfa blocklist",
			}
		}
		if blocked {
			return FraudCheckResult{
				Block:  true,
				Reason: "device_id_blocklist: idfa is in blocklist",
			}
		}
	}

	if input.GAID == "" && input.IDFA == "" {
		return FraudCheckResult{
			Block:  false,
			Reason: "device_id_blocklist: no device id provided",
		}
	}

	return FraudCheckResult{
		Block:  false,
		Reason: "device_id_blocklist: device id not in blocklist",
	}
}