package api

import (
	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"github.com/gofiber/fiber/v2"
)

func listFeedItem(c *fiber.Ctx) error {
	take := c.QueryInt("take", 10)
	offset := c.QueryInt("offset", 0)

	var count int64
	if err := database.C.Model(&models.SubscriptionItem{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var items []models.SubscriptionItem
	if err := database.C.
		Order("published_at DESC").
		Omit("Content").
		Preload("Feed").
		Limit(take).Offset(offset).Find(&items).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  items,
	})
}

func getFeedItem(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id", 0)

	var item models.SubscriptionItem
	if err := database.C.Where("id = ?", id).Preload("Feed").First(&item).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(item)
}
