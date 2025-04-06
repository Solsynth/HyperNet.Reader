package models

import (
	"crypto/md5"
	"encoding/hex"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"
	"github.com/google/uuid"
	"time"
)

type SubscriptionFeed struct {
	cruda.BaseModel

	URL           string     `json:"url"`
	IsEnabled     bool       `json:"is_enabled"`
	IsFullContent bool       `json:"is_full_content"`
	PullInterval  int        `json:"pull_interval"`
	Adapter       string     `json:"adapter"`
	AccountID     *uint      `json:"account_id"`
	LastFetchedAt *time.Time `json:"last_fetched_at"`
}

type SubscriptionItem struct {
	cruda.BaseModel

	FeedID      uint             `json:"feed_id"`
	Feed        SubscriptionFeed `json:"feed"`
	Thumbnail   string           `json:"thumbnail"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Content     string           `json:"content"`
	URL         string           `json:"url"`
	Hash        string           `json:"hash" gorm:"uniqueIndex"`

	// PublishedAt is the time when the article is published, when the feed adapter didn't provide this default to creation date
	PublishedAt time.Time `json:"published_at"`
}

func (v *SubscriptionItem) GenHash() {
	if len(v.URL) == 0 {
		v.URL = uuid.NewString()
		return
	}

	hash := md5.Sum([]byte(v.URL))
	v.Hash = hex.EncodeToString(hash[:])
}
