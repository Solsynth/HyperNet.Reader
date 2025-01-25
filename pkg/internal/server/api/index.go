package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"github.com/gofiber/fiber/v2"
)

func MapAPIs(app *fiber.App, baseURL string) {
	api := app.Group(baseURL).Name("API")
	{
		admin := api.Group("/admin").Name("Admin")
		{
			admin.Post("/scan", sec.ValidatorMiddleware, adminTriggerScanTask)
		}

		api.Get("/well-known/sources", getNewsSources)
		api.Get("/link/*", getLinkMeta)
	}
}
