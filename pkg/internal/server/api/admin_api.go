package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/reader/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func adminTriggerScanTask(c *fiber.Ctx) error {
	if err := sec.EnsureGrantedPerm(c, "AdminTriggerNewsScan", true); err != nil {
		return err
	}

	go services.ScanNewsSources()
	return c.SendStatus(fiber.StatusOK)
}
