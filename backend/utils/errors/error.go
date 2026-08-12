package errors

import "errors"

// auth errors
var (
	// ErrInvalidPasswordComplexity enforces nfr01 password policy
	ErrInvalidPasswordComplexity = errors.New("password must be between 6 and 8 characters long and contain at least one special character")
)

// team errors
var (
	ErrTeamNameRequired      = errors.New("team name is required")
	ErrTeamNameTooLong       = errors.New("team name must be 20 characters or less")
	ErrUnauthorizedTeamOp    = errors.New("unauthorized team operation: owner permission required")
	ErrSuperAdminRequired    = errors.New("unauthorized operation: super_admin scope required to delete a team")
	ErrUserNotInTeam         = errors.New("user is not a member of this team")
	ErrInvalidIncidentStatus = errors.New("invalid incident status: must be OPEN, IN_PROGRESS, RESOLVED, or CLOSED")
	ErrIncidentNotFound      = errors.New("incident not found")
	ErrTeamNotFound          = errors.New("team not found")
	ErrRunbookNotFound       = errors.New("runbook not found")
	ErrInstructionTooShort   = errors.New("instruction details must be at least 30 characters long")
	ErrUserNotFound          = errors.New("User not found")
)

// dashboard errors
var (
	ErrInvalidTimeframe      = errors.New("invalid timeframe: must be day, month, or year")
	ErrInvalidSLATarget      = errors.New("sla_target_minutes must be a positive integer")
	ErrDashboardUnauthorized = errors.New("unauthorized: must be a team member to access dashboard analytics")
)
