package types

import (
	"time"

	"github.com/google/uuid"
)

// UserWithTeamMembershipRow represents a flattened row from users LEFT JOIN team_members LEFT JOIN teams
type UserWithTeamMembershipRow struct {
	UserID         uuid.UUID  `gorm:"column:user_id"`
	FirebaseUID    *string    `gorm:"column:firebase_uid"`
	Username       *string    `gorm:"column:username"`
	Email          string     `gorm:"column:email"`
	DisplayName    string     `gorm:"column:display_name"`
	UserCreatedAt  time.Time  `gorm:"column:user_created_at"`
	DeactivatedAt  *time.Time `gorm:"column:deactivated_at"`
	Scope          string     `gorm:"column:scope"`
	MembershipID   *uuid.UUID `gorm:"column:membership_id"`
	TeamID         *uuid.UUID `gorm:"column:team_id"`
	MembershipRole *string    `gorm:"column:membership_role"`
	TeamName       *string    `gorm:"column:team_name"`
	TeamCreatedAt  *time.Time `gorm:"column:team_created_at"`
}

// TeamWithMemberRow represents a flattened row from teams LEFT JOIN team_members LEFT JOIN users
type TeamWithMemberRow struct {
	TeamID          uuid.UUID  `gorm:"column:team_id"`
	TeamName        string     `gorm:"column:team_name"`
	TeamCreatedAt   time.Time  `gorm:"column:team_created_at"`
	MemberID        *uuid.UUID `gorm:"column:member_id"`
	MemberUserID    *uuid.UUID `gorm:"column:member_user_id"`
	MemberRole      *string    `gorm:"column:member_role"`
	UserEmail       *string    `gorm:"column:user_email"`
	UserDisplayName *string    `gorm:"column:user_display_name"`
	UserScope       *string    `gorm:"column:user_scope"`
}

// TeamMemberWithUserRow represents a row from team_members JOIN users
type TeamMemberWithUserRow struct {
	ID              uuid.UUID `gorm:"column:id"`
	TeamID          uuid.UUID `gorm:"column:team_id"`
	UserID          uuid.UUID `gorm:"column:user_id"`
	Role            string    `gorm:"column:role"`
	UserEmail       string    `gorm:"column:user_email"`
	UserDisplayName string    `gorm:"column:user_display_name"`
	UserScope       string    `gorm:"column:user_scope"`
	UserTotpEnabled bool      `gorm:"column:user_totp_enabled"`
	UserCreatedAt   time.Time `gorm:"column:user_created_at"`
}

// TeamIncidentWithHistoryRow represents a flattened row from team_incidents LEFT JOIN incident_status_histories
type TeamIncidentWithHistoryRow struct {
	IncidentID            uuid.UUID  `gorm:"column:incident_id"`
	IncidentNumber        string     `gorm:"column:incident_number"`
	TeamID                uuid.UUID  `gorm:"column:team_id"`
	CreatedBy             uuid.UUID  `gorm:"column:created_by"`
	Title                 string     `gorm:"column:title"`
	Status                string     `gorm:"column:status"`
	Details               string     `gorm:"column:details"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	AssignedAt            time.Time  `gorm:"column:assigned_at"`
	ResolvedAt            *time.Time `gorm:"column:resolved_at"`
	HistoryID             *uuid.UUID `gorm:"column:history_id"`
	HistoryUpdatedBy      *uuid.UUID `gorm:"column:history_updated_by"`
	HistoryTitle          *string    `gorm:"column:history_title"`
	HistoryNewStatus      *string    `gorm:"column:history_new_status"`
	HistoryPreviousStatus *string    `gorm:"column:history_previous_status"`
	HistoryDetails        *string    `gorm:"column:history_details"`
	HistoryUpdatedAt      *time.Time `gorm:"column:history_updated_at"`
}

// ConversationWithUserRow represents a row from conversations LEFT JOIN users
type ConversationWithUserRow struct {
	ID              uuid.UUID  `gorm:"column:id"`
	TeamID          uuid.UUID  `gorm:"column:team_id"`
	TeamIncidentID  *uuid.UUID `gorm:"column:team_incident_id"`
	UserID          uuid.UUID  `gorm:"column:user_id"`
	Title           string     `gorm:"column:title"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UserEmail       *string    `gorm:"column:user_email"`
	UserDisplayName *string    `gorm:"column:user_display_name"`
	UserScope       *string    `gorm:"column:user_scope"`
}

// ConversationWithMessagesRow represents a row from conversations LEFT JOIN users LEFT JOIN messages
type ConversationWithMessagesRow struct {
	ConvID           uuid.UUID  `gorm:"column:conv_id"`
	TeamID           uuid.UUID  `gorm:"column:team_id"`
	TeamIncidentID   *uuid.UUID `gorm:"column:team_incident_id"`
	UserID           uuid.UUID  `gorm:"column:user_id"`
	Title            string     `gorm:"column:title"`
	ConvCreatedAt    time.Time  `gorm:"column:conv_created_at"`
	UserEmail        *string    `gorm:"column:user_email"`
	UserDisplayName  *string    `gorm:"column:user_display_name"`
	UserScope        *string    `gorm:"column:user_scope"`
	MessageID        *uuid.UUID `gorm:"column:message_id"`
	ParentMessageID  *uuid.UUID `gorm:"column:parent_message_id"`
	MessageSender    *string    `gorm:"column:message_sender"`
	MessageContent   *string    `gorm:"column:message_content"`
	MessageCreatedAt *time.Time `gorm:"column:message_created_at"`
}
