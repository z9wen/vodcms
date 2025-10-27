package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"vodcms/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase 初始化数据库
func InitDatabase() error {
	var err error

	// 配置日志
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// 连接SQLite数据库
	DB, err = gorm.Open(sqlite.Open("vodcms.db"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 自动迁移表结构
	err = DB.AutoMigrate(
		&models.Video{},
		&models.Source{},
		&models.CollectionLog{},
		&models.UnmappedCategory{},
		&models.MappingRule{},
		&models.FuzzyMatchRule{},
	)
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	fmt.Println("数据库初始化成功")
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
