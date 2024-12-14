package models

import "git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"

type LinkMeta struct {
	cruda.BaseModel

	Entry       string  `json:"entry_id" gorm:"uniqueIndex"`
	Icon        string  `json:"icon"`
	URL         string  `json:"url"`
	Title       *string `json:"title"`
	Image       *string `json:"image"`
	Video       *string `json:"video"`
	Audio       *string `json:"audio"`
	Description *string `json:"description"`
	SiteName    *string `json:"site_name"`
	Type        *string `json:"type"`
}
