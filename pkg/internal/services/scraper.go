package services

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.solsynth.dev/hypernet/reader/pkg/internal/database"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"github.com/gocolly/colly"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

// We have to set the User-Agent to this so the sites will respond with opengraph data
const ScrapLinkDefaultUA = "facebookexternalhit/1.1"

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

	ua := viper.GetString("scraper.expand_ua")
	if len(ua) == 0 {
		ua = ScrapLinkDefaultUA
	}

	c := colly.NewCollector(
		colly.UserAgent(ua),
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

const ScrapNewsDefaultUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1.1 Safari/605.1.15"

func ScrapSubscriptionFeed(target string, maxPages ...int) []models.SubscriptionItem {
	parsedTarget, err := url.Parse(target)
	if err != nil {
		return nil
	}
	baseUrl := fmt.Sprintf("%s://%s", parsedTarget.Scheme, parsedTarget.Host)

	ua := viper.GetString("scraper.news_ua")
	if len(ua) == 0 {
		ua = ScrapNewsDefaultUA
	}

	var limit int
	if len(maxPages) > 0 && maxPages[0] > 0 {
		limit = maxPages[0]
	}

	c := colly.NewCollector(
		colly.UserAgent(ua),
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

	var result []models.SubscriptionItem

	c.OnHTML("main a", func(e *colly.HTMLElement) {
		if limit <= 0 {
			return
		}

		url := e.Attr("href")
		if strings.HasPrefix(url, "#") || strings.HasPrefix(url, "javascript:") || strings.HasPrefix(url, "mailto:") {
			return
		}
		if !strings.HasPrefix(url, "http") {
			url = fmt.Sprintf("%s%s", baseUrl, url)
		}

		limit--
		article, err := ScrapSubscriptionItem(url)
		if err != nil {
			log.Warn().Err(err).Str("url", url).Msg("Failed to scrap a news article...")
			return
		}

		log.Debug().Str("url", url).Msg("Scraped a news article...")
		if article != nil {
			result = append(result, *article)
		}
	})

	_ = c.Visit(target)

	return result
}

func ScrapSubscriptionItem(target string, parent ...models.SubscriptionItem) (*models.SubscriptionItem, error) {
	ua := viper.GetString("scraper.news_ua")
	if len(ua) == 0 {
		ua = ScrapNewsDefaultUA
	}

	c := colly.NewCollector(
		colly.UserAgent(ua),
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

	article := &models.SubscriptionItem{
		URL: target,
	}

	if len(parent) > 0 {
		article.Content = parent[0].Content
		article.Thumbnail = parent[0].Thumbnail
		article.Description = parent[0].Description
	}

	c.OnHTML("title", func(e *colly.HTMLElement) {
		if len(article.Title) == 0 {
			article.Title = e.Text
		}
	})
	c.OnHTML("meta[name]", func(e *colly.HTMLElement) {
		switch e.Attr("name") {
		case "description":
			if len(article.Description) == 0 {
				article.Description = e.Attr("content")
			}
		}
	})

	c.OnHTML("article", func(e *colly.HTMLElement) {
		if len(article.Content) == 0 {
			article.Content, _ = e.DOM.Html()
		}
	})
	c.OnHTML("article img", func(e *colly.HTMLElement) {
		if len(article.Thumbnail) == 0 {
			url := e.Attr("src")
			// Usually, if the image have a relative path, it is some static assets instead of content.
			if strings.HasPrefix(url, "http") {
				article.Thumbnail = url
			}
		}
	})

	return article, c.Visit(target)
}
