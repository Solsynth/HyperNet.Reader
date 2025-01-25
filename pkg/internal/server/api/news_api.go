package api

import (
	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"github.com/gofiber/fiber/v2"
)

func listNewsArticles(c *fiber.Ctx) error {
	take := c.QueryInt("take", 0)
	offset := c.QueryInt("offset", 0)

	var count int64
	if err := database.C.Model(&models.NewsArticle{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var articles []models.NewsArticle
	if err := database.C.Limit(take).Offset(offset).
		Omit("Content").Order("created_at DESC").
		Find(&articles).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  articles,
	})
}

func getNewsArticle(c *fiber.Ctx) error {
	hash := c.Params("hash")

	var article models.NewsArticle
	if err := database.C.Where("hash = ?", hash).First(&article).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(article)
}
