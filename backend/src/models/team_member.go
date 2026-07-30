package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamMemberRole string

const (
	TeamRoleLeader TeamMemberRole = "leader"
	TeamRoleAdmin  TeamMemberRole = "admin"
	TeamRoleMember TeamMemberRole = "member"
)

type TeamMemberStatus string

const (
	MemberPending TeamMemberStatus = "pending"
	MemberActive  TeamMemberStatus = "active"
	MemberRemoved TeamMemberStatus = "removed"
)


type TeamMember struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Team Relationship
	TeamID uuid.UUID `gorm:"type:uuid;not null;index"`
	Team   Team      `gorm:"foreignKey:TeamID;constraint:OnDelete:CASCADE"`

	// User Relationship
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	// Member Details
	Role TeamMemberRole `gorm:"type:varchar(20);default:'member'"`
	Status TeamMemberStatus `gorm:"type:varchar(20);default:'pending'"`

	JoinedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (tm *TeamMember) BeforeCreate(tx *gorm.DB) error {
	tm.ID = uuid.New()
	return nil
}