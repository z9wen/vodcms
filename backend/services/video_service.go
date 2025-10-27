package services

import (
	"fmt"
	"time"

	"vodcms/config"
	"vodcms/handles"
	"vodcms/models"
	"vodcms/utils"
)

// VideoService 视频服务
type VideoService struct {
	collector       *handles.Collector
	sourceManager   *handles.SourceManager
	categoryMapping *CategoryMappingService
}

// NewVideoService 创建视频服务
func NewVideoService() *VideoService {
	db := config.GetDB()
	return &VideoService{
		collector:       handles.NewCollector(),
		sourceManager:   handles.NewSourceManager("sources_config.json"),
		categoryMapping: NewCategoryMappingService("category_mapping.json", db),
	}
}

// CollectAndSave 采集并保存视频到数据库
func (vs *VideoService) CollectAndSave(mode handles.CollectMode, sourceKeys []string, maxPages int) error {
	db := config.GetDB()

	// 加载数据源配置
	if err := vs.sourceManager.LoadSources(); err != nil {
		return fmt.Errorf("加载数据源失败: %w", err)
	}

	// 获取要采集的源
	var sources []handles.Source
	if len(sourceKeys) == 0 {
		sources = vs.sourceManager.GetEnabledSources()
	} else {
		allSources := vs.sourceManager.GetEnabledSources()
		for _, source := range allSources {
			for _, key := range sourceKeys {
				if source.Key == key {
					sources = append(sources, source)
					break
				}
			}
		}
	}

	if len(sources) == 0 {
		return fmt.Errorf("没有可用的数据源")
	}

	// 采集每个源
	for _, source := range sources {
		startTime := time.Now()

		// 记录采集日志
		log := models.CollectionLog{
			SourceName: source.Name,
			SourceKey:  source.Key,
			Mode:       string(mode),
			StartTime:  startTime,
			Status:     "running",
		}
		db.Create(&log)

		// 执行采集
		stats := vs.collector.CollectSource(source, mode, maxPages)

		// 更新日志
		log.TotalPages = stats.TotalPages
		log.TotalVideos = stats.TotalVideos
		log.SuccessCount = stats.SuccessCount
		log.ErrorCount = stats.ErrorCount
		log.Duration = stats.Duration
		log.EndTime = time.Now()

		if stats.ErrorCount > 0 {
			log.Status = "partial"
		} else if stats.SuccessCount > 0 {
			log.Status = "success"
		} else {
			log.Status = "failed"
		}

		db.Save(&log)

		// 读取采集的JSON文件并保存到数据库
		if err := utils.ImportVideoFromJSON(source.Key); err != nil {
			fmt.Printf("⚠️ 导入数据库失败: %v\n", err)
		}
	}

	return nil
}

// SyncSourcesToDB 同步数据源配置到数据库
func (vs *VideoService) SyncSourcesToDB() error {
	db := config.GetDB()

	if err := vs.sourceManager.LoadSources(); err != nil {
		return err
	}

	sources := vs.sourceManager.GetDefaultSources()

	for _, s := range sources {
		var dbSource models.Source
		result := db.Where("key = ?", s.Key).First(&dbSource)

		if result.RowsAffected == 0 {
			// 不存在则创建
			dbSource = models.Source{
				Name:    s.Name,
				BaseURL: s.BaseURL,
				Key:     s.Key,
				Enabled: s.Enabled,
			}
			db.Create(&dbSource)
			fmt.Printf("✅ 已添加数据源: %s\n", s.Name)
		} else {
			// 存在则更新
			dbSource.Name = s.Name
			dbSource.BaseURL = s.BaseURL
			dbSource.Enabled = s.Enabled
			db.Save(&dbSource)
		}
	}

	return nil
}
