package handles

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"vodcms/config"
	"vodcms/models"
	"vodcms/utils"
)

// CollectVideos 触发视频采集
func CollectVideos(c *gin.Context) {
	var req struct {
		Mode       string   `json:"mode"`        // today, week, month, all
		SourceKeys []string `json:"source_keys"` // 要采集的源
		MaxPages   int      `json:"max_pages"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "无效的请求数据",
		})
		return
	}

	// 转换模式
	var mode CollectMode
	switch req.Mode {
	case "today":
		mode = CollectToday
	case "week":
		mode = CollectWeek
	case "month":
		mode = CollectMonth
	case "all":
		mode = CollectAll
	default:
		mode = CollectToday
	}

	// 异步执行采集任务
	go func() {
		if err := collectAndSave(mode, req.SourceKeys, req.MaxPages); err != nil {
			fmt.Printf("❌ 采集失败: %v\n", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "采集任务已启动",
		"data": gin.H{
			"mode":        req.Mode,
			"source_keys": req.SourceKeys,
			"started_at":  time.Now(),
		},
	})
}

// collectAndSave 内部采集并保存函数
func collectAndSave(mode CollectMode, sourceKeys []string, maxPages int) error {
	db := config.GetDB()
	collector := NewCollector()
	sourceManager := NewSourceManager("sources_config.json")

	// 加载数据源配置
	if err := sourceManager.LoadSources(); err != nil {
		return fmt.Errorf("加载数据源失败: %w", err)
	}

	// 获取要采集的源
	var sources []Source
	if len(sourceKeys) == 0 {
		sources = sourceManager.GetEnabledSources()
	} else {
		allSources := sourceManager.GetEnabledSources()
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
		stats := collector.CollectSource(source, mode, maxPages)

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

// GetCollectionLogs 获取采集日志
func GetCollectionLogs(c *gin.Context) {
	db := config.GetDB()

	var logs []models.CollectionLog
	result := db.Order("created_at DESC").Limit(50).Find(&logs)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": logs,
	})
}

// ImportJSON 导入JSON文件到数据库
func ImportJSON(c *gin.Context) {
	var req struct {
		SourceKey string `json:"source_key"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "无效的请求数据",
		})
		return
	}

	// TODO: 调用导入功能
	// utils.ImportVideoFromJSON(req.SourceKey)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "导入成功",
	})
}
