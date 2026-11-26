package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MilestoneAssignment links an accepted party member (or the solo freelancer)
// to a milestone. One row per (milestone, member); re-assignment replaces rows.
type MilestoneAssignment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	MilestoneID uuid.UUID      `gorm:"type:uuid;not null;index;uniqueIndex:uniq_milestone_member,priority:1" json:"milestoneId"`
	Milestone   QuestMilestone `gorm:"foreignKey:MilestoneID;constraint:OnDelete:CASCADE" json:"-"`

	MemberID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:uniq_milestone_member,priority:2" json:"memberId"`
	Member   User      `gorm:"foreignKey:MemberID;constraint:OnDelete:CASCADE" json:"-"`

	AssignedByID uuid.UUID `gorm:"type:uuid;not null" json:"assignedById"`

	StartDate *time.Time `json:"startDate"`
	EndDate   *time.Time `json:"endDate"`
	Note      string     `gorm:"type:text" json:"note"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (a *MilestoneAssignment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
