package services

import (
	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
)

func GetTodayFeedRandomly(limit int) ([]models.SubscriptionItem, error) {
	var articles []models.SubscriptionItem
	if err := database.C.Limit(limit).
		Where("DATE(created_at) = CURRENT_DATE"). // Created in today
		Order("RANDOM()").
		Omit("Content").
		Preload("Feed").
		Find(&articles).Error; err != nil {
		return articles, err
	}
	return articles, nil
}
