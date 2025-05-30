package database

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"
	"git.solsynth.dev/hypernet/reader/pkg/internal/gap"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var C *gorm.DB

func NewGorm() error {
	var err error

	dsn, err := cruda.NewCrudaConn(gap.Nx).AllocDatabase("reader")
	C, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.New(&log.Logger, logger.Config{
		Colorful:                  true,
		IgnoreRecordNotFoundError: true,
		LogLevel:                  lo.Ternary(viper.GetBool("debug.database"), logger.Info, logger.Silent),
	})})

	return err
}
