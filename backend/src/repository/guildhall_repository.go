package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

type GuildhallRepository struct{}

func NewGuildhallRepository() *GuildhallRepository {
	return &GuildhallRepository{}
}

func (r *GuildhallRepository) CreatePost(post *models.GuildhallPost) error {
	return postgres.DB.Create(post).Error
}

func (r *GuildhallRepository) FindPostByID(postID uuid.UUID) (*models.GuildhallPost, error) {
	var post models.GuildhallPost

	err := postgres.DB.
		Preload("Author").
		First(&post, "id = ?", postID).Error
	if err != nil {
		return nil, err
	}

	return &post, nil
}

func (r *GuildhallRepository) ListPosts(postType string, offset, limit int) ([]models.GuildhallPost, int64, error) {
	query := postgres.DB.Model(&models.GuildhallPost{})
	if postType != "" {
		query = query.Where("type = ?", postType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var posts []models.GuildhallPost
	err := query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (r *GuildhallRepository) UpdatePost(post *models.GuildhallPost) error {
	result := postgres.DB.
		Model(&models.GuildhallPost{}).
		Where("id = ? AND author_id = ?", post.ID, post.AuthorID).
		Updates(map[string]interface{}{
			"title":    post.Title,
			"content":  post.Content,
			"link_url": post.LinkURL,
			"details":  post.Details,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *GuildhallRepository) DeletePost(postID uuid.UUID) error {
	result := postgres.DB.Delete(&models.GuildhallPost{}, "id = ?", postID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
