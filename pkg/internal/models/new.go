package models

import (
	"crypto/md5"
	"encoding/hex"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"
	"github.com/google/uuid"
)

type NewsArticle struct {
	cruda.BaseModel

	Thumbnail   string `json:"thumbnail"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	URL         string `json:"url"`
	Hash        string `json:"hash" gorm:"uniqueIndex"`
	Source      string `json:"source"`
}

func (v *NewsArticle) GenHash() *NewsArticle {
	if len(v.URL) == 0 {
		v.Hash = uuid.NewString()
		return v
	}

	hash := md5.Sum([]byte(v.URL))
	v.Hash = hex.EncodeToString(hash[:])
	return v
}
