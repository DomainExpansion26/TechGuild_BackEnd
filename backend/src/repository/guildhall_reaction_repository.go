package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

type GuildhallReactionSummary struct {
	LikeCount int64 `gorm:"column:like_count"`
	LikedByMe bool  `gorm:"column:liked_by_me"`
}

type guildhallReactionSummaryRow struct {
	PostID    uuid.UUID `gorm:"column:post_id"`
	LikeCount int64     `gorm:"column:like_count"`
	LikedByMe bool      `gorm:"column:liked_by_me"`
}

type GuildhallReactionRepository struct{}

func NewGuildhallReactionRepository() *GuildhallReactionRepository {
	return &GuildhallReactionRepository{}
}

func (r *GuildhallReactionRepository) Like(postID, userID uuid.UUID) error {
	reaction := models.GuildhallReaction{
		PostID: postID,
		UserID: userID,
	}
	return postgres.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "post_id"},
			{Name: "user_id"},
		},
		DoNothing: true,
	}).Create(&reaction).Error
}

func (r *GuildhallReactionRepository) Unlike(postID, userID uuid.UUID) error {
	return postgres.DB.
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&models.GuildhallReaction{}).Error
}

func (r *GuildhallReactionRepository) Summaries(
	postIDs []uuid.UUID,
	userID uuid.UUID,
) (map[uuid.UUID]GuildhallReactionSummary, error) {
	summaries := make(map[uuid.UUID]GuildhallReactionSummary, len(postIDs))
	if len(postIDs) == 0 {
		return summaries, nil
	}

	var rows []guildhallReactionSummaryRow
	err := postgres.DB.Model(&models.GuildhallReaction{}).
		Select("post_id, COUNT(*) AS like_count, BOOL_OR(user_id = ?) AS liked_by_me", userID).
		Where("post_id IN ?", postIDs).
		Group("post_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		summaries[row.PostID] = GuildhallReactionSummary{
			LikeCount: row.LikeCount,
			LikedByMe: row.LikedByMe,
		}
	}
	return summaries, nil
}
