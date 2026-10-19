package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

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

type QuestComplexity string

const (
	QuestComplexityBeginner     QuestComplexity = "beginner"
	QuestComplexityIntermediate QuestComplexity = "intermediate"
	QuestComplexityAdvanced     QuestComplexity = "advanced"
	QuestComplexityEnterprise   QuestComplexity = "enterprise"
)

type QuestType string

const (
	QuestTypeStandard   QuestType = "standard"
	QuestTypeEmergency  QuestType = "emergency"
	QuestTypeLongTerm   QuestType = "long_term"
	QuestTypeMultiSkill QuestType = "multi_skill"
)

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
	BudgetINR       float64   `gorm:"not null"`
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

	// Assigned Freelancer / Agency
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

type QuestMilestone struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	QuestID uuid.UUID `gorm:"type:uuid;index;not null"`

	Title       string          `gorm:"type:varchar(255);not null"`
	Description string          `gorm:"type:text"`
	Amount      float64         `gorm:"not null"`
	DueDays     int             `gorm:"not null"`
	OrderIndex  int             `gorm:"not null"`
	Status      MilestoneStatus `gorm:"type:varchar(30);default:'pending'"`

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

