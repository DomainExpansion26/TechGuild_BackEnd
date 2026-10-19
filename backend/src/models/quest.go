package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ============================================================
// Quest Status
// ============================================================

type QuestStatus string

const (
	QuestStatusDraft      QuestStatus = "draft"
	QuestStatusOpen       QuestStatus = "open"
	QuestStatusPublished  QuestStatus = "published"
	QuestStatusInProgress QuestStatus = "in_progress"
	QuestStatusCompleted  QuestStatus = "completed"
	QuestStatusCancelled  QuestStatus = "cancelled"
	QuestStatusRejected   QuestStatus = "rejected"
)

// ============================================================
// Quest Complexity
// ============================================================

type QuestComplexity string

const (
	QuestComplexityBeginner     QuestComplexity = "beginner"
	QuestComplexityIntermediate QuestComplexity = "intermediate"
	QuestComplexityAdvanced     QuestComplexity = "advanced"
	QuestComplexityEnterprise   QuestComplexity = "enterprise"
)

// ============================================================
// Quest Type
// ============================================================

type QuestType string

const (
	QuestTypeStandard   QuestType = "standard"
	QuestTypeEmergency  QuestType = "emergency"
	QuestTypeLongTerm   QuestType = "long_term"
	QuestTypeMultiSkill QuestType = "multi_skill"
)

// ============================================================
// Milestone Status — Freelancer creates, Client reviews
// ============================================================

type MilestoneStatus string

const (
	// Freelancer is creating/editing
	MilestoneStatusDraft MilestoneStatus = "draft"

	// Freelancer sent to client, awaiting review
	MilestoneStatusAwaitingClientReview MilestoneStatus = "awaiting_client_review"

	// Client requested changes, freelancer needs to respond
	MilestoneStatusRevisionRequested MilestoneStatus = "revision_requested"

	// Freelancer is applying requested changes
	MilestoneStatusRevisionInProgress MilestoneStatus = "revision_in_progress"

	// Both parties agreed, ready to fund
	MilestoneStatusAgreed MilestoneStatus = "agreed"

	// Funding initiated but provider awaiting confirmation (Razorpay checkout)
	MilestoneStatusFundingPending MilestoneStatus = "funding_pending"

	// Client funded, escrow holds money
	MilestoneStatusFunded MilestoneStatus = "funded"

	// Freelancer is working on tasks
	MilestoneStatusWorkInProgress MilestoneStatus = "work_in_progress"

	// Freelancer submitted for release, awaiting client
	MilestoneStatusSubmittedForRelease MilestoneStatus = "submitted_for_release"

	// Client released, money in freelancer wallet
	MilestoneStatusReleased MilestoneStatus = "released"

	// Dispute raised
	MilestoneStatusDisputed MilestoneStatus = "disputed"

	// Cancelled
	MilestoneStatusCancelled MilestoneStatus = "cancelled"

	// Refund flow
	MilestoneStatusRefundPending MilestoneStatus = "refund_pending"
	MilestoneStatusRefunded      MilestoneStatus = "refunded"
)

// ============================================================
// Quest Model — created by CLIENT
// ============================================================

type Quest struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Quest Owner (Client)
	ClientID uuid.UUID `gorm:"type:uuid;index;not null"`
	Client   User      `gorm:"foreignKey:ClientID;constraint:OnDelete:CASCADE"`

	// Basic Information (Step 2)
	Title       string          `gorm:"type:varchar(100);not null"`
	Description string          `gorm:"type:text;not null"`
	Category    string          `gorm:"type:varchar(100);not null"`
	Industry    string          `gorm:"type:varchar(100)"`
	Complexity  QuestComplexity `gorm:"type:varchar(50);default:'intermediate'"`

	// Budget & Timeline (Step 3)
	// BudgetMinor is a CLIENT-SET reference amount in minor units (paise).
	// It does NOT automatically change when freelancer adds milestones.
	BudgetMinor     int64     `gorm:"not null;default:0"`
	TimelineDays    int       `gorm:"not null"`
	QuestType       QuestType `gorm:"type:varchar(50);default:'standard'"`
	RequiredMinRank string    `gorm:"type:varchar(10);not null"`

	// Requirements & Skills (Step 4)
	ExperienceLevel string         `gorm:"type:varchar(50)"`
	SoftSkills      pq.StringArray `gorm:"type:text[]"`
	Languages       datatypes.JSON `gorm:"type:jsonb"`

	// Status & Lifecycle
	Status      QuestStatus `gorm:"type:varchar(30);default:'draft'"`
	PublishedAt *time.Time

	// Escrow & Funding
	IsEscrowFunded   bool       `gorm:"default:false"`
	EscrowFundedAt   *time.Time
	EscrowRefundedAt *time.Time

	// Assigned Freelancer (after quest is accepted)
	AssignedFreelancerID *uuid.UUID `gorm:"type:uuid;index"`
	AssignedFreelancer   *User      `gorm:"foreignKey:AssignedFreelancerID"`

	// Accepted Party (Party Collaboration)
	AcceptedPartyID *uuid.UUID `gorm:"type:uuid;index"`
	AcceptedParty   *Party     `gorm:"foreignKey:AcceptedPartyID"`

	// Associations
	Milestones []QuestMilestone `gorm:"foreignKey:QuestID;constraint:OnDelete:CASCADE"`
	Skills     []QuestSkill     `gorm:"foreignKey:QuestID;constraint:OnDelete:CASCADE"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (q *Quest) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}

// ============================================================
// Quest Milestone — created by FREELANCER (after quest accept)
// ============================================================

type QuestMilestone struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	QuestID uuid.UUID `gorm:"type:uuid;index;not null"`

	// Freelancer who created this milestone
	CreatedByID uuid.UUID `gorm:"type:uuid;index;not null"`
	CreatedBy   User      `gorm:"foreignKey:CreatedByID"`

	Title       string          `gorm:"type:varchar(255);not null"`
	Description string          `gorm:"type:text"`
	AmountMinor int64           `gorm:"not null;default:0"`
	DueDays     int             `gorm:"not null"`
	OrderIndex  int             `gorm:"not null"`
	Status      MilestoneStatus `gorm:"type:varchar(40);default:'draft'"`

	// Revision tracking
	RevisionCount  int `gorm:"default:0"`
	LastRevisionAt *time.Time
	ClientFeedback string `gorm:"type:text"`

	// Lifecycle timestamps
	AgreedAt    *time.Time
	FundedAt    *time.Time
	ReleasedAt  *time.Time
	StartDate   *time.Time
	DueDate     *time.Time
	CompletedAt *time.Time
	ApprovedAt  *time.Time

	Tasks []Task `gorm:"foreignKey:MilestoneID;constraint:OnDelete:CASCADE"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (m *QuestMilestone) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// ============================================================
// Quest Skill — created by CLIENT
// ============================================================

type QuestSkill struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	QuestID uuid.UUID `gorm:"type:uuid;index;not null"`

	Name             string `gorm:"type:varchar(100);not null"`
	ProficiencyLevel string `gorm:"type:varchar(50);default:'intermediate'"`
	IsCore           bool   `gorm:"default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *QuestSkill) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type QuestChatChannel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	QuestID   uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"questId"`
	Quest     Quest      `gorm:"foreignKey:QuestID;constraint:OnDelete:CASCADE" json:"-"`
	PartyID   *uuid.UUID `gorm:"type:uuid;index" json:"partyId,omitempty"`
	Party     *Party     `gorm:"foreignKey:PartyID" json:"party,omitempty"`
	Name      string     `gorm:"type:varchar(255);not null" json:"name"`
	Type      string     `gorm:"type:varchar(50);default:'party_client'" json:"type"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

func (c *QuestChatChannel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

