package api

import (
	"git.solsynth.dev/hypernet/reader/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func getNewsSources(c *fiber.Ctx) error {
	return c.JSON(services.NewsSources)
}
