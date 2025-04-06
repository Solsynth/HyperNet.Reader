package services

import (
	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"time"
)

func GetTodayFeedRandomly(limit int, cursor *time.Time) ([]models.SubscriptionItem, error) {
	tx := database.C
	if cursor != nil {
		tx = tx.Where("published_at < ?", *cursor)
	}

	var articles []models.SubscriptionItem
	if err := tx.Limit(limit).
		Order("published_at DESC").
		Omit("Content").
		Preload("Feed").
		Find(&articles).Error; err != nil {
		return articles, err
	}
	return articles, nil
}
