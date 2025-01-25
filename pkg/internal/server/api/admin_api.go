package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/reader/pkg/internal/server/exts"
	"git.solsynth.dev/hypernet/reader/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func adminTriggerScanTask(c *fiber.Ctx) error {
	if err := sec.EnsureGrantedPerm(c, "AdminTriggerNewsScan", true); err != nil {
		return err
	}

	var data struct {
		Eager bool `json:"eager"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	go services.ScanNewsSources(data.Eager)
	return c.SendStatus(fiber.StatusOK)
}
