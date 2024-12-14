package database

import (
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"gorm.io/gorm"
)

var AutoMaintainRange = []any{
	&models.LinkMeta{},
}

func RunMigration(source *gorm.DB) error {
	if err := source.AutoMigrate(
		AutoMaintainRange...,
	); err != nil {
		return err
	}

	return nil
}
