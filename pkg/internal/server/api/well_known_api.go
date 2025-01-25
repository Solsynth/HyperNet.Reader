package api

import (
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"git.solsynth.dev/hypernet/reader/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

func getNewsSources(c *fiber.Ctx) error {
	return c.JSON(lo.Filter(services.NewsSources, func(item models.NewsSource, index int) bool {
		return item.Enabled
	}))
}
