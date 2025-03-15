package api

import (
	"time"

	"git.solsynth.dev/hypernet/reader/pkg/internal/services"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"github.com/gofiber/fiber/v2"
)

func getTodayNews(c *fiber.Ctx) error {
	tx := database.C
	today := time.Now().Format("2006-01-02")
	tx = tx.Where("DATE(COALESCE(published_at, created_at)) = ?", today)

	var count int64
	countTx := tx
	if err := countTx.Model(&models.NewsArticle{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var article models.NewsArticle
	if err := tx.
		Omit("Content").Order("COALESCE(published_at, created_at) DESC").
		First(&article).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  article,
	})
}

func listNewsArticles(c *fiber.Ctx) error {
	if err := sec.EnsureGrantedPerm(c, "ListNews", true); err != nil {
		return err
	}

	take := c.QueryInt("take", 0)
	offset := c.QueryInt("offset", 0)
	source := c.Query("source")

	tx := database.C

	if len(source) > 0 {
		tx = tx.Where("source = ?", source)
	}

	isAdvanced := false
	if err := sec.EnsureGrantedPerm(c, "ListNewsAdvanced", true); err == nil {
		isAdvanced = true
	}

	var sources []string
	for _, srv := range services.GetNewsSources() {
		if !isAdvanced && srv.Advanced {
			continue
		}
		sources = append(sources, srv.ID)
	}

	tx = tx.Where("source IN ?", sources)

	var count int64
	countTx := tx
	if err := countTx.Model(&models.NewsArticle{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var articles []models.NewsArticle
	if err := tx.Limit(take).Offset(offset).
		Omit("Content").Order("COALESCE(published_at, created_at) DESC").
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
