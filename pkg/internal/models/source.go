package models

type NewsSource struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Type   string `json:"type"`
	Source string `json:"source"`
	Depth  int    `json:"depth"`
}
