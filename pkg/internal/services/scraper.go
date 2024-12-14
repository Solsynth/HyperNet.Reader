package services

import (
	"crypto/md5"
	"encoding/hex"
	"net"
	"net/http"
	"time"

	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"github.com/gocolly/colly"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// We have to set the User-Agent to this so the sites will respond with opengraph data
const ScrapLinkUserAgent = "facebookexternalhit/1.1"

func GetLinkMetaFromCache(target string) (models.LinkMeta, error) {
	hash := md5.Sum([]byte(target))
	entry := hex.EncodeToString(hash[:])
	var meta models.LinkMeta
	if err := database.C.Where("entry = ?", entry).First(&meta).Error; err != nil {
		return meta, err
	}
	return meta, nil
}

func SaveLinkMetaToCache(target string, meta models.LinkMeta) error {
	hash := md5.Sum([]byte(target))
	entry := hex.EncodeToString(hash[:])
	meta.Entry = entry
	return database.C.Save(&meta).Error
}

func ScrapLink(target string) (*models.LinkMeta, error) {
	if cache, err := GetLinkMetaFromCache(target); err == nil {
		log.Debug().Str("url", target).Msg("Expanding link... hit cache")
		return &cache, nil
	}

	c := colly.NewCollector(
		colly.UserAgent(ScrapLinkUserAgent),
		colly.MaxDepth(3),
	)

	c.WithTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 360 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	})

	meta := &models.LinkMeta{
		URL: target,
	}

	c.OnHTML("title", func(e *colly.HTMLElement) {
		meta.Title = &e.Text
	})
	c.OnHTML("meta[name]", func(e *colly.HTMLElement) {
		switch e.Attr("name") {
		case "description":
			meta.Description = lo.ToPtr(e.Attr("content"))
		}
	})
	c.OnHTML("meta[property]", func(e *colly.HTMLElement) {
		switch e.Attr("property") {
		case "og:title":
			meta.Title = lo.ToPtr(e.Attr("content"))
		case "og:description":
			meta.Description = lo.ToPtr(e.Attr("content"))
		case "og:image":
			meta.Image = lo.ToPtr(e.Attr("content"))
		case "og:video":
			meta.Video = lo.ToPtr(e.Attr("content"))
		case "og:audio":
			meta.Audio = lo.ToPtr(e.Attr("content"))
		case "og:site_name":
			meta.SiteName = lo.ToPtr(e.Attr("content"))
		case "og:type":
			meta.Type = lo.ToPtr(e.Attr("content"))
		}
	})
	c.OnHTML("link[rel]", func(e *colly.HTMLElement) {
		if e.Attr("rel") == "icon" {
			meta.Icon = e.Request.AbsoluteURL(e.Attr("href"))
		}
	})

	c.OnRequest(func(r *colly.Request) {
		log.Debug().Str("url", target).Msg("Scraping link... requesting")
	})
	c.RedirectHandler = func(req *http.Request, via []*http.Request) error {
		log.Debug().Str("url", req.URL.String()).Msg("Scraping link... redirecting")
		return nil
	}

	c.OnResponse(func(r *colly.Response) {
		log.Debug().Str("url", target).Msg("Scraping link... analyzing")
	})
	c.OnError(func(r *colly.Response, err error) {
		log.Warn().Err(err).Str("url", target).Msg("Scraping link... failed")
	})

	c.OnScraped(func(r *colly.Response) {
		_ = SaveLinkMetaToCache(target, *meta)
		log.Debug().Str("url", target).Msg("Scraping link... finished")
	})

	return meta, c.Visit(target)
}
