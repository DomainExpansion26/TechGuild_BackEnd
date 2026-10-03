package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PartyMemberRole string

const (
	PartyRoleLeader PartyMemberRole = "leader"
	PartyRoleAdmin  PartyMemberRole = "admin"
	PartyRoleMember PartyMemberRole = "member"
)

type PartyMemberStatus string

const (
	MemberPending PartyMemberStatus = "pending"
	MemberActive  PartyMemberStatus = "active"
	MemberRemoved PartyMemberStatus = "removed"
)


type PartyMember struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Party Relationship
	PartyID uuid.UUID `gorm:"type:uuid;not null;index"`
	Party   Party      `gorm:"foreignKey:PartyID;constraint:OnDelete:CASCADE"`

	// User Relationship
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	// Member Details
	Role PartyMemberRole `gorm:"type:varchar(20);default:'member'"`
	Status PartyMemberStatus `gorm:"type:varchar(20);default:'pending'"`

	JoinedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (tm *PartyMember) BeforeCreate(tx *gorm.DB) error {
	tm.ID = uuid.New()
	return nil
}