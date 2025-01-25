package api

import (
	"github.com/gofiber/fiber/v2"
)

func MapAPIs(app *fiber.App, baseURL string) {
	api := app.Group(baseURL).Name("API")
	{
		api.Get("/well-known/sources", getNewsSources)
		api.Get("/link/*", getLinkMeta)
	}
}
