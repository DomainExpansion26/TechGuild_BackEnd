package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TaskStatus string

const (
	TaskStatusToDo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
)

// Task represents an individual task within a quest milestone.
type Task struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	MilestoneID uuid.UUID      `gorm:"type:uuid;index;not null" json:"milestoneId"`
	Milestone   QuestMilestone `gorm:"foreignKey:MilestoneID;constraint:OnDelete:CASCADE" json:"-"`

	QuestID uuid.UUID `gorm:"type:uuid;index;not null" json:"questId"`
	Quest   Quest     `gorm:"foreignKey:QuestID;constraint:OnDelete:CASCADE" json:"-"`

	Title       string     `gorm:"type:varchar(255);not null" json:"title"`
	Description string     `gorm:"type:text" json:"description"`
	AssigneeID  *uuid.UUID `gorm:"type:uuid;index" json:"assigneeId"`
	Assignee    *User      `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`

	StartDate *time.Time `json:"startDate"`
	EndDate   *time.Time `json:"endDate"`

	Status TaskStatus `gorm:"type:varchar(30);default:'todo'" json:"status"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type ActivityActionType string

const (
	ActivityQuestAssigned              ActivityActionType = "quest_assigned"
	ActivityTaskCreated                ActivityActionType = "task_created"
	ActivityTaskStatusChanged          ActivityActionType = "task_status_changed"
	ActivityMilestoneCreated           ActivityActionType = "milestone_created"
	ActivityMilestoneDeleted           ActivityActionType = "milestone_deleted"
	ActivityMilestoneApproved          ActivityActionType = "milestone_approved"
	ActivityMilestoneApprovalCancelled ActivityActionType = "milestone_approval_cancelled"
	ActivityQuestApproved              ActivityActionType = "quest_approved"
	ActivityQuestApprovalCancelled     ActivityActionType = "quest_approval_cancelled"
)

// QuestActivity records chronological audit events for the TMD activity feed and drawer.
type QuestActivity struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	QuestID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"questId"`
	MilestoneID *uuid.UUID `gorm:"type:uuid;index" json:"milestoneId,omitempty"`
	TaskID      *uuid.UUID `gorm:"type:uuid;index" json:"taskId,omitempty"`

	ActorID uuid.UUID `gorm:"type:uuid;not null;index" json:"actorId"`
	Actor   User      `gorm:"foreignKey:ActorID" json:"actor"`

	ActionType  ActivityActionType `gorm:"type:varchar(50);not null" json:"actionType"`
	Title       string             `gorm:"type:varchar(255);not null" json:"title"`
	Description string             `gorm:"type:text" json:"description"`
	Metadata    datatypes.JSON     `gorm:"type:jsonb" json:"metadata,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

func (a *QuestActivity) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// QuestReview stores client rating and feedback when completing/approving a quest.
type QuestReview struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	QuestID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"questId"`
	ClientID  uuid.UUID `gorm:"type:uuid;not null;index" json:"clientId"`
	Rating    int       `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Feedback  string    `gorm:"type:text;not null" json:"feedback"`
	CreatedAt time.Time `json:"createdAt"`
}

func (r *QuestReview) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// MilestoneApprovalLog records justification and timestamps when milestone approval is toggled.
type MilestoneApprovalLog struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	MilestoneID uuid.UUID `gorm:"type:uuid;not null;index" json:"milestoneId"`
	ClientID    uuid.UUID `gorm:"type:uuid;not null;index" json:"clientId"`
	Action      string    `gorm:"type:varchar(30);not null" json:"action"` // "approve" or "cancel"
	Reason      string    `gorm:"type:text" json:"reason"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (m *MilestoneApprovalLog) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
