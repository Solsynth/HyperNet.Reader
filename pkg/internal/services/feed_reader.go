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
	"gorm.io/gorm/clause"
)

func FetchFeedTimed() {
	FetchFeed(false)
}

func FetchFeed(eager ...bool) {
	var feeds []models.SubscriptionFeed
	if len(eager) > 0 && eager[0] {
		if err := database.C.Where("is_enabled = ?", true).Find(&feeds).Error; err != nil {
			log.Warn().Err(err).Msg("An error occurred when fetching feeds.")
			return
		}
	} else {
		if err := database.C.
			Where("last_fetched_at IS NULL OR NOW() >= last_fetched_at + (pull_interval || ' hours')::interval").
			Find(&feeds).Error; err != nil {
			log.Warn().Err(err).Msg("An error occurred when fetching due feeds.")
			return
		}
	}

	log.Info().Int("count", len(feeds)).Msg("Ready to fetch feeds...")

	count := 0
	var scannedFeed []uint
	for _, src := range feeds {
		if !src.IsEnabled {
			continue
		}

		log.Debug().Uint("source", src.ID).Msg("Scanning feed...")
		result, err := SubscriptionFeedRead(src, eager...)
		if err != nil {
			log.Warn().Err(err).Uint("source", src.ID).Msg("Failed to scan a feed.")
		} else {
			scannedFeed = append(scannedFeed, src.ID)
		}

		result = lo.UniqBy(result, func(item models.SubscriptionItem) string {
			return item.Hash
		})
		database.C.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "hash"}},
			DoUpdates: clause.AssignmentColumns([]string{"thumbnail", "title", "content", "description", "published_at"}),
		}).Create(&result)

		log.Info().Uint("source", src.ID).Int("count", len(result)).Msg("Scanned a feed.")
		count += len(result)
	}

	database.C.
		Model(&models.SubscriptionFeed{}).
		Where("id IN ?", scannedFeed).
		Update("last_fetched_at", time.Now())

	log.Info().Int("count", count).Msg("Scanned all feeds.")
}

func SubscriptionFeedRead(src models.SubscriptionFeed, eager ...bool) ([]models.SubscriptionItem, error) {
	switch src.Adapter {
	case "wordpress":
		return feedReadWordpress(src, eager...)
	case "webpage":
		return feedReadWebpage(src, eager...)
	case "feed":
		return feedReadGuidedFeed(src, eager...)
	default:
		return nil, fmt.Errorf("unsupported feed source type: %s", src.Adapter)
	}
}

func feedReadWordpress(src models.SubscriptionFeed, eager ...bool) ([]models.SubscriptionItem, error) {
	wpConvert := func(post wordpress.Post) models.SubscriptionItem {
		article := &models.SubscriptionItem{
			Title:       post.Title.Rendered,
			Description: post.Excerpt.Rendered,
			Content:     post.Content.Rendered,
			URL:         post.Link,
			FeedID:      src.ID,
		}
		date, err := time.Parse("2006-01-02T15:04:05", post.DateGMT)
		if err == nil {
			article.PublishedAt = date
		} else {
			article.PublishedAt = time.Now()
		}
		article.GenHash()
		return *article
	}

	client := wordpress.NewClient(&wordpress.Options{
		BaseAPIURL: src.URL,
	})

	posts, resp, _, err := client.Posts().List(nil)
	if err != nil {
		return nil, err
	}

	var result []models.SubscriptionItem
	for _, post := range posts {
		result = append(result, wpConvert(post))
	}

	if len(eager) > 0 && eager[0] {
		totalPagesRaw := resp.Header.Get("X-WP-TotalPages")
		totalPages, _ := strconv.Atoi(totalPagesRaw)
		depth := min(totalPages, 10)
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

func feedReadGuidedFeed(src models.SubscriptionFeed, eager ...bool) ([]models.SubscriptionItem, error) {
	pgConvert := func(article models.SubscriptionItem) models.SubscriptionItem {
		art := &article
		art.GenHash()
		art.FeedID = src.ID
		article = *art
		return article
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURLWithContext(src.URL, ctx)

	maxPages := lo.TernaryF(len(eager) > 0 && eager[0], func() int {
		if feed.Items == nil {
			return 0
		}
		return len(feed.Items)
	}, func() int {
		return 10 * 10
	})

	var result []models.SubscriptionItem
	for _, item := range feed.Items {
		if maxPages <= 0 {
			break
		}

		maxPages--
		parent := models.SubscriptionItem{
			URL:         item.Link,
			Title:       item.Title,
			Description: item.Description,
		}
		if item.PublishedParsed != nil {
			parent.PublishedAt = *item.PublishedParsed
		} else {
			parent.PublishedAt = time.Now()
		}
		if item.Image != nil {
			parent.Thumbnail = item.Image.URL
		}

		article, err := ScrapSubscriptionItem(item.Link, parent)
		if err != nil {
			log.Warn().Err(err).Str("url", item.Link).Msg("Failed to scrap a news article...")
			continue
		}
		result = append(result, pgConvert(*article))

		log.Debug().Str("url", item.Link).Msg("Scraped a news article...")
	}

	return result, nil
}

func feedReadWebpage(src models.SubscriptionFeed, eager ...bool) ([]models.SubscriptionItem, error) {
	pgConvert := func(article models.SubscriptionItem) models.SubscriptionItem {
		art := &article
		art.GenHash()
		art.FeedID = src.ID
		art.PublishedAt = time.Now()
		article = *art
		return article
	}

	maxPages := lo.Ternary(len(eager) > 0 && eager[0], 0, 10*10)
	result := ScrapSubscriptionFeed(src.URL, maxPages)

	for idx, page := range result {
		result[idx] = pgConvert(page)
	}

	return result, nil
}
