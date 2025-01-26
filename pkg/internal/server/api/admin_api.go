package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"git.solsynth.dev/hypernet/reader/pkg/internal/server/exts"
	"git.solsynth.dev/hypernet/reader/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"gorm.io/gorm/clause"
)

func adminTriggerScanTask(c *fiber.Ctx) error {
	if err := sec.EnsureGrantedPerm(c, "AdminTriggerNewsScan", true); err != nil {
		return err
	}

	var data struct {
		Eager   bool     `json:"eager"`
		Sources []string `json:"sources"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	go func() {
		count := 0
		for _, src := range services.NewsSources {
			if !src.Enabled {
				continue
			}
			if len(data.Sources) > 0 && !lo.Contains(data.Sources, src.ID) {
				continue
			}

			log.Debug().Str("source", src.ID).Msg("Scanning news source...")
			result, err := services.NewsSourceRead(src, data.Eager)
			if err != nil {
				log.Warn().Err(err).Str("source", src.ID).Msg("Failed to scan a news source.")
			}

			result = lo.UniqBy(result, func(item models.NewsArticle) string {
				return item.Hash
			})
			database.C.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "hash"}},
				DoUpdates: clause.AssignmentColumns([]string{"thumbnail", "title", "content", "description", "published_at"}),
			}).Create(&result)

			log.Info().Str("source", src.ID).Int("count", len(result)).Msg("Scanned a news sources.")
			count += len(result)
		}

		log.Info().Int("count", count).Msg("Scanned all news sources.")
	}()

	return c.SendStatus(fiber.StatusOK)
}
