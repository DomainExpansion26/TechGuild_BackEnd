package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

type GuildhallVoteCount struct {
	Option string `gorm:"column:option"`
	Votes  int64  `gorm:"column:votes"`
}

type GuildhallVoteRepository struct{}

func NewGuildhallVoteRepository() *GuildhallVoteRepository {
	return &GuildhallVoteRepository{}
}

func (r *GuildhallVoteRepository) Cast(postID, userID uuid.UUID, option string) error {
	vote := models.GuildhallVote{
		PostID: postID,
		UserID: userID,
		Option: option,
	}
	return postgres.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "post_id"},
			{Name: "user_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"option", "updated_at"}),
	}).Create(&vote).Error
}

func (r *GuildhallVoteRepository) HasVotes(postID uuid.UUID) (bool, error) {
	var count int64
	err := postgres.DB.Model(&models.GuildhallVote{}).
		Where("post_id = ?", postID).
		Count(&count).Error
	return count > 0, err
}

func (r *GuildhallVoteRepository) HasUserVoted(postID, userID uuid.UUID) (bool, error) {
	var count int64
	err := postgres.DB.Model(&models.GuildhallVote{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *GuildhallVoteRepository) UserOption(postID, userID uuid.UUID) (string, error) {
	var vote models.GuildhallVote
	result := postgres.DB.
		Select("option").
		Where("post_id = ? AND user_id = ?", postID, userID).
		First(&vote)
	if result.Error != nil {
		return "", result.Error
	}
	return vote.Option, nil
}

func (r *GuildhallVoteRepository) Counts(postID uuid.UUID) ([]GuildhallVoteCount, int64, error) {
	var counts []GuildhallVoteCount
	err := postgres.DB.Model(&models.GuildhallVote{}).
		Select(`"option", COUNT(*) AS votes`).
		Where("post_id = ?", postID).
		Group(`"option"`).
		Scan(&counts).Error
	if err != nil {
		return nil, 0, err
	}

	var total int64
	for _, count := range counts {
		total += count.Votes
	}
	return counts, total, nil
}
