package services

import (
	"fmt"
	"strconv"

	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/sogko/go-wordpress"
	"github.com/spf13/viper"
	"gorm.io/gorm/clause"
)

var NewsSources []models.NewsSource

func LoadNewsSources() error {
	if err := viper.UnmarshalKey("sources", &NewsSources); err != nil {
		return err
	}
	log.Info().Int("count", len(NewsSources)).Msg("Loaded news sources configuration.")
	return nil
}

func ScanNewsSourcesNoEager() {
	ScanNewsSources(false)
}

func ScanNewsSources(eager ...bool) {
	var results []models.NewsArticle
	for _, src := range NewsSources {
		if !src.Enabled {
			continue
		}

		log.Debug().Str("source", src.ID).Msg("Scanning news source...")
		result, err := NewsSourceRead(src)
		if err != nil {
			log.Warn().Err(err).Str("source", src.ID).Msg("Failed to scan a news source.")
		}
		results = append(results, result...)
		log.Info().Str("source", src.ID).Int("count", len(result)).Msg("Scanned a news sources.")
	}
	log.Info().Int("count", len(results)).Msg("Scanned all news sources.")

	results = lo.UniqBy(results, func(item models.NewsArticle) string {
		return item.Hash
	})

	database.C.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&results)
}

func NewsSourceRead(src models.NewsSource, eager ...bool) ([]models.NewsArticle, error) {
	switch src.Type {
	case "wordpress":
		return newsSourceReadWordpress(src, eager...)
	case "scrap":
		return newsSourceReadScrap(src)
	default:
		return nil, fmt.Errorf("unsupported news source type: %s", src.Type)
	}
}

func newsSourceReadWordpress(src models.NewsSource, eager ...bool) ([]models.NewsArticle, error) {
	wpConvert := func(post wordpress.Post) models.NewsArticle {
		article := &models.NewsArticle{
			Title:       post.Title.Rendered,
			Description: post.Excerpt.Rendered,
			Content:     post.Content.Rendered,
			URL:         post.Link,
			Source:      src.ID,
		}
		article.GenHash()
		return *article
	}

	client := wordpress.NewClient(&wordpress.Options{
		BaseAPIURL: src.Source,
	})

	posts, resp, _, err := client.Posts().List(nil)
	if err != nil {
		return nil, err
	}

	var result []models.NewsArticle
	for _, post := range posts {
		result = append(result, wpConvert(post))
	}

	if len(eager) > 0 && eager[0] {
		totalPagesRaw := resp.Header.Get("X-WP-TotalPages")
		totalPages, _ := strconv.Atoi(totalPagesRaw)
		depth := min(totalPages, src.Depth)
		for page := 2; page <= depth; page++ {
			posts, _, _, err := client.Posts().List(nil)
			if err != nil {
				return result, nil
			}
			for _, post := range posts {
				result = append(result, wpConvert(post))
			}
		}
	}

	return result, nil
}

func newsSourceReadScrap(src models.NewsSource) ([]models.NewsArticle, error) {
	pgConvert := func(article models.NewsArticle) models.NewsArticle {
		art := &article
		art.GenHash()
		art.Source = src.ID
		article = *art
		return article
	}

	result := ScrapNewsIndex(src.Source)

	for idx, page := range result {
		result[idx] = pgConvert(page)
	}

	return result, nil
}
