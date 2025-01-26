package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/mmcdole/gofeed"
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
	count := 0
	for _, src := range NewsSources {
		if !src.Enabled {
			continue
		}

		log.Debug().Str("source", src.ID).Msg("Scanning news source...")
		result, err := NewsSourceRead(src, eager...)
		if err != nil {
			log.Warn().Err(err).Str("source", src.ID).Msg("Failed to scan a news source.")
		}

		result = lo.UniqBy(result, func(item models.NewsArticle) string {
			return item.Hash
		})
		database.C.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "hash"}},
			DoUpdates: clause.Assignments(map[string]interface{}{}),
		}).Create(&result)

		log.Info().Str("source", src.ID).Int("count", len(result)).Msg("Scanned a news sources.")
		count += len(result)
	}

	log.Info().Int("count", count).Msg("Scanned all news sources.")
}

func NewsSourceRead(src models.NewsSource, eager ...bool) ([]models.NewsArticle, error) {
	switch src.Type {
	case "wordpress":
		return newsSourceReadWordpress(src, eager...)
	case "scrap":
		return newsSourceReadScrap(src, eager...)
	case "feed":
		return newsSourceReadFeed(src, eager...)
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
		time, err := time.Parse("2006-01-02T15:04:05", post.DateGMT)
		if err == nil {
			article.PublishedAt = &time
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
			posts, _, _, err := client.Posts().List(fiber.Map{
				"page": page,
			})
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

func newsSourceReadFeed(src models.NewsSource, eager ...bool) ([]models.NewsArticle, error) {
	pgConvert := func(article models.NewsArticle) models.NewsArticle {
		art := &article
		art.GenHash()
		art.Source = src.ID
		article = *art
		return article
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURLWithContext(src.Source, ctx)

	maxPages := lo.Ternary(len(eager) > 0 && eager[0], len(feed.Items), src.Depth)

	var result []models.NewsArticle
	for _, item := range feed.Items {
		if maxPages <= 0 {
			break
		}

		maxPages--
		parent := models.NewsArticle{
			URL:         item.Link,
			Title:       item.Title,
			Description: item.Description,
		}
		if item.PublishedParsed != nil {
			parent.PublishedAt = item.PublishedParsed
		}
		if item.Image != nil {
			parent.Thumbnail = item.Image.URL
		}

		article, err := ScrapNews(item.Link, parent)
		if err != nil {
			log.Warn().Err(err).Str("url", item.Link).Msg("Failed to scrap a news article...")
			continue
		}
		result = append(result, pgConvert(*article))

		log.Debug().Str("url", item.Link).Msg("Scraped a news article...")
	}

	return result, nil
}

func newsSourceReadScrap(src models.NewsSource, eager ...bool) ([]models.NewsArticle, error) {
	pgConvert := func(article models.NewsArticle) models.NewsArticle {
		art := &article
		art.GenHash()
		art.Source = src.ID
		article = *art
		return article
	}

	maxPages := lo.Ternary(len(eager) > 0 && eager[0], 0, src.Depth)
	result := ScrapNewsIndex(src.Source, maxPages)

	for idx, page := range result {
		result[idx] = pgConvert(page)
	}

	return result, nil
}
