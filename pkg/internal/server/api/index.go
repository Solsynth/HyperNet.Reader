package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/reader/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func MapAPIs(app *fiber.App, baseURL string) {
	api := app.Group(baseURL).Name("API")
	{
		api.Get("/well-known/sources", getNewsSources)

		admin := api.Group("/admin").Name("Admin")
		{
			admin.Get("/scan", func(c *fiber.Ctx) error {
				services.ScanNewsSources()
				return c.SendStatus(fiber.StatusOK)
			})

			admin.Post("/scan", sec.ValidatorMiddleware, adminTriggerScanTask)
		}

		api.Get("/link/*", getLinkMeta)

		news := api.Group("/news").Name("News")
		{
			news.Get("/", listNewsArticles)
			news.Get("/:hash", getNewsArticle)
		}
	}
}
