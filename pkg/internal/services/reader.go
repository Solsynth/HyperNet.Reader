package services

import (
	"fmt"
	"strconv"

	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"github.com/rs/zerolog/log"
	"github.com/sogko/go-wordpress"
	"github.com/spf13/viper"
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
		log.Debug().Str("source", src.ID).Msg("Scanning news source...")
		result, err := NewsSourceRead(src)
		if err != nil {
			log.Warn().Err(err).Str("source", src.ID).Msg("Failed to scan a news source.")
		}
		results = append(results, result...)
		log.Info().Str("source", src.ID).Int("count", len(result)).Msg("Scanned a news sources.")
	}
	log.Info().Int("count", len(results)).Msg("Scanned all news sources.")
	database.C.Save(&results)
}

func NewsSourceRead(src models.NewsSource, eager ...bool) ([]models.NewsArticle, error) {
	switch src.Type {
	case "wordpress":
		return newsSourceReadWordpress(src)
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
