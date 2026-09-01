package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID    `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	FirebaseUID   *string      `gorm:"type:varchar(255);unique;default:null" json:"firebase_uid,omitempty"`
	Username      *string      `gorm:"type:varchar(255);unique;default:null" json:"username,omitempty"`
	PasswordHash  string       `gorm:"type:varchar(255)" json:"-"`
	TOTPSecret    string       `gorm:"type:varchar(255)" json:"-"`
	TOTPEnabled   bool         `gorm:"default:false" json:"totp_enabled"`
	Email         string       `gorm:"type:varchar(255);not null;unique" json:"email"`
	DisplayName   string       `gorm:"type:varchar(100)" json:"display_name"`
	CreatedAt     time.Time    `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP" json:"created_at"`
	DeactivatedAt *time.Time   `gorm:"type:timestamp(0)" json:"deactivated_at,omitempty"`
	Scope         string       `gorm:"type:varchar(50);not null" json:"scope"`
	Memberships   []TeamMember `gorm:"foreignKey:UserID" json:"memberships,omitempty"`
}
