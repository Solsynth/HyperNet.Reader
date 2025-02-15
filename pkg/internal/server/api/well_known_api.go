package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"git.solsynth.dev/hypernet/reader/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

func getNewsSources(c *fiber.Ctx) error {
	isAdvanced := false
	if err := sec.EnsureGrantedPerm(c, "ListNewsAdvanced", true); err == nil {
		isAdvanced = true
	}

	return c.JSON(lo.Filter(services.NewsSources, func(item models.NewsSource, index int) bool {
		if !isAdvanced && item.Advanced {
			return false
		}
		return item.Enabled
	}))
}
