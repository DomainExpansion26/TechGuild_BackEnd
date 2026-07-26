package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectStatus string

const (
	ProjectDraft     ProjectStatus = "draft"
	ProjectPublished ProjectStatus = "published"
	ProjectClosed    ProjectStatus = "closed"
	ProjectCompleted ProjectStatus = "completed"
	ProjectArchived  ProjectStatus = "archived"
)

type BudgetType string

const (
	BudgetFixed BudgetType = "fixed"
	BudgetHourly BudgetType = "hourly"
)

type ProjectVisibility string

const (
	VisibilityPublic  ProjectVisibility = "public"
	VisibilityPrivate ProjectVisibility = "private"
)

type Project struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Project Owner
	ClientID uuid.UUID `gorm:"type:uuid;index;not null"`
	Client   User      `gorm:"foreignKey:ClientID;constraint:OnDelete:CASCADE"`

	// Basic Information
	Title       string `gorm:"size:255;not null"`
	Description string `gorm:"type:text;not null"`
	Category    string `gorm:"size:100;not null"`

	// Budget
	BudgetType BudgetType `gorm:"type:varchar(20);not null"`
	MinBudget  float64
	MaxBudget  float64
	Currency   string `gorm:"size:10;default:'INR'"`

	// Requirements
	ExperienceLevel string         `gorm:"size:50"`
	ProjectType     string         `gorm:"size:50"`
	Duration        string         `gorm:"size:100"`
	// Skills
	Skills []ProjectSkill `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	// Attachments
	Attachments []ProjectAttachment `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	// Applications related to the project by agency and freelancers
	Applications []ProjectApplication `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	// Visibility
	Visibility ProjectVisibility `gorm:"type:varchar(20);default:'public'"`

	// Status
	Status ProjectStatus `gorm:"type:varchar(30);default:'draft'"`

	// Dates
	ApplicationDeadline *time.Time
	EstimatedStartDate  *time.Time
	EstimatedEndDate    *time.Time
	PublishedAt         *time.Time

	// Limits
	MaxApplications int `gorm:"default:0"`

	// Highlights
	IsFeatured bool `gorm:"default:false"`
	IsUrgent   bool `gorm:"default:false"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *Project) BeforeCreate(tx *gorm.DB) error {
	p.ID = uuid.New()
	return nil
}