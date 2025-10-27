package handles

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"vodcms/config"
	"vodcms/models"
)

// GetVideos 获取视频列表（列表页去重，每个视频只显示一个版本）
func GetVideos(c *gin.Context) {
	db := config.GetDB()

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取筛选参数
	sourceKey := c.Query("source_key")
	typeName := c.Query("type_name")
	area := c.Query("area")
	keyword := c.Query("keyword")

	// 方案：使用子查询获取每个vod_id的最新记录（按采集时间）
	// 这样列表页每个视频只显示一次，但保留了所有源的数据在数据库中
	subQuery := db.Table("videos").
		Select("MAX(id) as id").
		Group("vod_id")

	// 构建主查询
	query := db.Model(&models.Video{}).
		Where("id IN (?)", subQuery)

	if sourceKey != "" {
		query = query.Where("source_key = ?", sourceKey)
	}
	if typeName != "" {
		query = query.Where("type_name = ?", typeName)
	}
	if area != "" {
		query = query.Where("vod_area = ?", area)
	}
	if keyword != "" {
		query = query.Where("vod_name LIKE ?", "%"+keyword+"%")
	}

	// 获取总数（去重后的）
	var total int64
	query.Count(&total)

	// 分页查询
	var videos []models.Video
	offset := (page - 1) * pageSize
	result := query.Order("collected_at DESC").Limit(pageSize).Offset(offset).Find(&videos)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result.Error.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"list":      videos,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetVideoByID 获取单个视频详情（包含所有源的播放地址）
func GetVideoByID(c *gin.Context) {
	db := config.GetDB()

	// 从URL中获取ID或vod_id
	id := c.Query("id")
	vodID := c.Query("vod_id")

	if id == "" && vodID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "ID参数缺失",
		})
		return
	}

	// 获取主视频信息
	var mainVideo models.Video
	if id != "" {
		// 通过数据库ID查询
		result := db.First(&mainVideo, id)
		if result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"code": 404,
				"msg":  "视频不存在",
			})
			return
		}
	} else {
		// 通过vod_id查询（取最新的一条）
		result := db.Where("vod_id = ?", vodID).Order("collected_at DESC").First(&mainVideo)
		if result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"code": 404,
				"msg":  "视频不存在",
			})
			return
		}
	}

	// 查询该视频在所有源中的版本（用于提供多个播放源）
	var allSources []models.Video
	db.Where("vod_id = ?", mainVideo.VodID).Order("collected_at DESC").Find(&allSources)

	// 构建播放源列表
	var playSources []map[string]interface{}
	for _, source := range allSources {
		playSources = append(playSources, map[string]interface{}{
			"source_key":    source.SourceKey,
			"source_name":   source.SourceName,
			"vod_play_url":  source.VodPlayURL,
			"vod_play_from": source.VodPlayFrom,
			"collected_at":  source.CollectedAt,
			"vod_remarks":   source.VodRemarks,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"video":        mainVideo,
			"play_sources": playSources,
			"source_count": len(allSources),
		},
	})
}

// GetVideoStats 获取视频统计信息
func GetVideoStats(c *gin.Context) {
	db := config.GetDB()

	var totalCount int64
	db.Model(&models.Video{}).Count(&totalCount)

	// 按来源统计
	var sourceCounts []struct {
		SourceKey  string
		SourceName string
		Count      int64
	}
	db.Model(&models.Video{}).
		Select("source_key, source_name, COUNT(*) as count").
		Group("source_key, source_name").
		Scan(&sourceCounts)

	// 按分类统计
	var typeCounts []struct {
		TypeName string
		Count    int64
	}
	db.Model(&models.Video{}).
		Select("type_name, COUNT(*) as count").
		Group("type_name").
		Order("count DESC").
		Limit(20).
		Scan(&typeCounts)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"total":         totalCount,
			"source_counts": sourceCounts,
			"type_counts":   typeCounts,
		},
	})
}

// GetVideoPlayURL 获取视频播放地址
// GET /api/videos/play?vod_id=xxx&source_key=xxx
func GetVideoPlayURL(c *gin.Context) {
	db := config.GetDB()

	vodID := c.Query("vod_id")
	sourceKey := c.Query("source_key") // 可选，指定特定源

	if vodID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "vod_id参数缺失",
		})
		return
	}

	// 构建查询
	query := db.Where("vod_id = ?", vodID)
	if sourceKey != "" {
		query = query.Where("source_key = ?", sourceKey)
	}

	// 查询所有匹配的视频源
	var videos []models.Video
	result := query.Order("collected_at DESC").Find(&videos)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result.Error.Error(),
		})
		return
	}

	if len(videos) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "未找到播放地址",
		})
		return
	}

	// 构建播放源列表
	type PlaySource struct {
		SourceKey    string `json:"source_key"`
		SourceName   string `json:"source_name"`
		PlayFrom     string `json:"play_from"`     // 播放来源标识（如m3u8, mp4等）
		PlayURL      string `json:"play_url"`      // 播放URL列表
		PlayServer   string `json:"play_server"`   // 播放服务器
		PlayNote     string `json:"play_note"`     // 播放说明
		DownFrom     string `json:"down_from"`     // 下载来源
		DownURL      string `json:"down_url"`      // 下载地址
		VodRemarks   string `json:"vod_remarks"`   // 备注（如更新状态）
		CollectedAt  string `json:"collected_at"`  // 采集时间
		Quality      string `json:"quality"`       // 画质标识
	}

	var playSources []PlaySource
	for _, video := range videos {
		source := PlaySource{
			SourceKey:   video.SourceKey,
			SourceName:  video.SourceName,
			PlayFrom:    video.VodPlayFrom,
			PlayURL:     video.VodPlayURL,
			PlayServer:  video.VodPlayServer,
			PlayNote:    video.VodPlayNote,
			DownFrom:    video.VodDownFrom,
			DownURL:     video.VodDownURL,
			VodRemarks:  video.VodRemarks,
			CollectedAt: video.CollectedAt.Format("2006-01-02 15:04:05"),
		}

		// 简单判断画质
		if video.SourceKey == "snzy" {
			source.Quality = "高清"
		} else if video.SourceKey == "hhzy" {
			source.Quality = "标清"
		} else {
			source.Quality = "标准"
		}

		playSources = append(playSources, source)
	}

	// 返回基本视频信息 + 播放源列表
	mainVideo := videos[0]
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"vod_id":       mainVideo.VodID,
			"vod_name":     mainVideo.VodName,
			"vod_pic":      mainVideo.VodPic,
			"type_name":    mainVideo.TypeName,
			"vod_remarks":  mainVideo.VodRemarks,
			"vod_actor":    mainVideo.VodActor,
			"vod_director": mainVideo.VodDirector,
			"vod_year":     mainVideo.VodYear,
			"vod_area":     mainVideo.VodArea,
			"vod_content":  mainVideo.VodContent,
			"play_sources": playSources,
			"source_count": len(playSources),
		},
	})
}
