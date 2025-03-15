package services

import (
	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
)

func GetTodayNewsRandomly(limit int, isAdvanced bool) ([]models.NewsArticle, error) {
	var sources []string
	for _, srv := range GetNewsSources() {
		if !isAdvanced && srv.Advanced {
			continue
		}
		sources = append(sources, srv.ID)
	}

	var articles []models.NewsArticle
	if err := database.C.Limit(limit).
		Where("source IN ?", sources).
		Where("DATE(created_at) = CURRENT_DATE"). // Created in today
		Order("RANDOM()").
		Find(&articles).Error; err != nil {
		return articles, err
	}
	return articles, nil
}
