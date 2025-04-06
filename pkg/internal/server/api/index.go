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

		api.Get("/link/*", getLinkMeta)

		subscription := api.Group("/subscriptions").Name("Subscriptions")
		{
			subscription.Get("/", listFeedItem)
			subscription.Get("/:id", getFeedItem)

			feed := subscription.Group("/feed").Name("Feed")
			{
				feed.Get("/", listFeedSubscriptions)
				feed.Get("/me", listCreatedFeedSubscriptions)
				feed.Get("/:id", getFeedSubscription)
				feed.Post("/", createFeedSubscription)
				feed.Put("/:id", updateFeedSubscription)
				feed.Post("/:id/toggle", toggleFeedSubscription)
				feed.Delete("/:id", deleteFeedSubscription)
			}
		}
	}
}
