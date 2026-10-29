package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuildhallCommentRepository struct{}

func NewGuildhallCommentRepository() *GuildhallCommentRepository {
	return &GuildhallCommentRepository{}
}

func (r *GuildhallCommentRepository) Create(comment *models.GuildhallComment) error {
	return postgres.DB.Create(comment).Error
}

func (r *GuildhallCommentRepository) FindByID(commentID uuid.UUID) (*models.GuildhallComment, error) {
	var comment models.GuildhallComment
	err := postgres.DB.First(&comment, "id = ?", commentID).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *GuildhallCommentRepository) ListByPostID(
	postID uuid.UUID,
	offset int,
	limit int,
) ([]models.GuildhallComment, int64, error) {
	query := postgres.DB.Model(&models.GuildhallComment{}).Where("post_id = ?", postID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var comments []models.GuildhallComment
	err := query.
		Order("created_at ASC").
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (r *GuildhallCommentRepository) Update(comment *models.GuildhallComment) error {
	result := postgres.DB.
		Model(&models.GuildhallComment{}).
		Where("id = ? AND author_id = ?", comment.ID, comment.AuthorID).
		Update("content", comment.Content)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *GuildhallCommentRepository) Delete(commentID uuid.UUID) error {
	result := postgres.DB.Delete(&models.GuildhallComment{}, "id = ?", commentID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
