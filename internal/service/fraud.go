package service

import (
	"context"

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
			NewUserIDBlocklistCheck(queries),
			NewGAIDBlocklistCheck(queries),
			NewIDFABlocklistCheck(queries),
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

type UserIDBlocklistCheck struct {
	queries *db.Queries
}

func NewUserIDBlocklistCheck(queries *db.Queries) *UserIDBlocklistCheck {
	return &UserIDBlocklistCheck{queries: queries}
}

func (c *UserIDBlocklistCheck) Name() string {
	return "user_id_blocklist"
}

func (c *UserIDBlocklistCheck) Check(ctx context.Context, input TrackInput, clickID string) FraudCheckResult {
	if input.UserID == "" {
		return FraudCheckResult{
			Block:  false,
			Reason: "user_id not provided",
		}
	}

	blocked, err := c.queries.IsBlocked(ctx, input.UserID)
	if err != nil {
		return FraudCheckResult{
			Block:  false,
			Reason: "error checking blocklist",
		}
	}

	if blocked {
		return FraudCheckResult{
			Block:  true,
			Reason: "user_id is in blocklist",
		}
	}

	return FraudCheckResult{
		Block:  false,
		Reason: "user_id not in blocklist",
	}
}

type GAIDBlocklistCheck struct {
	queries *db.Queries
}

func NewGAIDBlocklistCheck(queries *db.Queries) *GAIDBlocklistCheck {
	return &GAIDBlocklistCheck{queries: queries}
}

func (c *GAIDBlocklistCheck) Name() string {
	return "gaid_blocklist"
}

func (c *GAIDBlocklistCheck) Check(ctx context.Context, input TrackInput, clickID string) FraudCheckResult {
	if input.GAID == "" {
		return FraudCheckResult{
			Block:  false,
			Reason: "gaid not provided",
		}
	}

	blocked, err := c.queries.IsBlocked(ctx, input.GAID)
	if err != nil {
		return FraudCheckResult{
			Block:  false,
			Reason: "error checking blocklist",
		}
	}

	if blocked {
		return FraudCheckResult{
			Block:  true,
			Reason: "gaid is in blocklist",
		}
	}

	return FraudCheckResult{
		Block:  false,
		Reason: "gaid not in blocklist",
	}
}

type IDFABlocklistCheck struct {
	queries *db.Queries
}

func NewIDFABlocklistCheck(queries *db.Queries) *IDFABlocklistCheck {
	return &IDFABlocklistCheck{queries: queries}
}

func (c *IDFABlocklistCheck) Name() string {
	return "idfa_blocklist"
}

func (c *IDFABlocklistCheck) Check(ctx context.Context, input TrackInput, clickID string) FraudCheckResult {
	if input.IDFA == "" {
		return FraudCheckResult{
			Block:  false,
			Reason: "idfa not provided",
		}
	}

	blocked, err := c.queries.IsBlocked(ctx, input.IDFA)
	if err != nil {
		return FraudCheckResult{
			Block:  false,
			Reason: "error checking blocklist",
		}
	}

	if blocked {
		return FraudCheckResult{
			Block:  true,
			Reason: "idfa is in blocklist",
		}
	}

	return FraudCheckResult{
		Block:  false,
		Reason: "idfa not in blocklist",
	}
}