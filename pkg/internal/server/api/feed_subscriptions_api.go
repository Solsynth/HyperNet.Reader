package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"git.solsynth.dev/hypernet/reader/pkg/internal/server/exts"
	"github.com/gofiber/fiber/v2"
)

func listFeedSubscriptions(c *fiber.Ctx) error {
	take := c.QueryInt("take", 10)
	offset := c.QueryInt("offset", 0)

	var count int64
	if err := database.C.Model(&models.SubscriptionFeed{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var feeds []models.SubscriptionFeed
	if err := database.C.Limit(take).Offset(offset).Find(&feeds).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  feeds,
	})
}

func listCreatedFeedSubscriptions(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("nex_user").(*sec.UserInfo)

	take := c.QueryInt("take", 10)
	offset := c.QueryInt("offset", 0)

	tx := database.C.Where("account_id = ?", user.ID)

	var count int64
	countTx := tx
	if err := countTx.Model(&models.SubscriptionFeed{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var feeds []models.SubscriptionFeed
	if err := tx.Take(take).Offset(offset).Find(&feeds).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  feeds,
	})
}

func getFeedSubscription(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id", 0)

	var feed models.SubscriptionFeed
	if err := database.C.Where("id = ?", id).First(&feed).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(feed)
}

func createFeedSubscription(c *fiber.Ctx) error {
	if err := sec.EnsureGrantedPerm(c, "CreateFeedSubscription", true); err != nil {
		return err
	}
	user := c.Locals("nex_user").(*sec.UserInfo)

	var data struct {
		URL          string `json:"url" validate:"required,url"`
		PullInterval int    `json:"pull_interval" validate:"required,min=6,max=720"`
		Adapter      string `json:"adapter"`
	}
	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	feed := models.SubscriptionFeed{
		URL:          data.URL,
		PullInterval: data.PullInterval,
		Adapter:      data.Adapter,
		AccountID:    &user.ID,
	}

	if err := database.C.Create(&feed).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(feed)
}

func updateFeedSubscription(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("nex_user").(*sec.UserInfo)

	id, _ := c.ParamsInt("id", 0)

	var data struct {
		URL          string `json:"url" validate:"required,url"`
		PullInterval int    `json:"pull_interval" validate:"required,min=6,max=720"`
		Adapter      string `json:"adapter"`
	}
	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var feed models.SubscriptionFeed
	if err := database.C.Where("account_id = ? AND id = ?", user.ID, id).First(&feed).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	feed.URL = data.URL
	feed.PullInterval = data.PullInterval
	feed.Adapter = data.Adapter

	if err := database.C.Save(&feed).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(feed)
}

func toggleFeedSubscription(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("nex_user").(*sec.UserInfo)

	id, _ := c.ParamsInt("id", 0)

	var feed models.SubscriptionFeed
	if err := database.C.Where("account_id = ? AND id = ?", user.ID, id).First(&feed).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	feed.IsEnabled = !feed.IsEnabled

	if err := database.C.Save(&feed).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(feed)
}

func deleteFeedSubscription(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("nex_user").(*sec.UserInfo)

	id, _ := c.ParamsInt("id", 0)

	var feed models.SubscriptionFeed
	if err := database.C.Where("account_id = ? AND id = ?", user.ID, id).First(&feed).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := database.C.Delete(&feed).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}
