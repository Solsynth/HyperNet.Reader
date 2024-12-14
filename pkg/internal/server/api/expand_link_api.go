package api

import (
	"encoding/base64"
	"sync"

	"git.solsynth.dev/hypernet/reader/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

var expandInProgress sync.Map

func getLinkMeta(c *fiber.Ctx) error {
	targetEncoded := c.Params("*1")
	targetRaw, _ := base64.StdEncoding.DecodeString(targetEncoded)

	if ch, loaded := expandInProgress.LoadOrStore(targetEncoded, make(chan struct{})); loaded {
		// If the request is already in progress, wait for it to complete
		<-ch.(chan struct{})
	} else {
		// If this is the first request, process it and signal others
		defer func() {
			close(ch.(chan struct{}))
			expandInProgress.Delete(targetEncoded)
		}()
	}

	if meta, err := services.ScrapLink(string(targetRaw)); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(meta)
	}
}
