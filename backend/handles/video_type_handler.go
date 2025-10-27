package handles

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"vodcms/config"
	"vodcms/models"
)

// GetVideoTypes 获取分类列表
func GetVideoTypes(c *gin.Context) {
	db := config.GetDB()

	// 获取筛选参数
	sourceKey := c.Query("source_key")
	isActive := c.Query("is_active")
	unifiedName := c.Query("unified_name")

	query := db.Model(&models.VideoType{})

	if sourceKey != "" {
		query = query.Where("source_key = ?", sourceKey)
	}
	if isActive != "" {
		active := isActive == "true" || isActive == "1"
		query = query.Where("is_active = ?", active)
	}
	if unifiedName != "" {
		query = query.Where("unified_name = ?", unifiedName)
	}

	var types []models.VideoType
	result := query.Order("sort ASC, type_name ASC").Find(&types)

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
		"data": types,
	})
}

// GetVideoTypeStats 获取分类统计（每个分类下有多少视频）
func GetVideoTypeStats(c *gin.Context) {
	db := config.GetDB()

	// 按分类统计视频数量
	var typeStats []struct {
		TypeName string `json:"type_name"`
		Count    int64  `json:"count"`
	}

	// 使用子查询去重后再统计
	subQuery := db.Table("videos").
		Select("MAX(id) as id, type_name").
		Group("vod_id")

	db.Table("(?) as deduplicated", subQuery).
		Select("type_name, COUNT(*) as count").
		Group("type_name").
		Order("count DESC").
		Scan(&typeStats)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": typeStats,
	})
}

// UpdateVideoType 更新分类信息（主要用于设置unified_name）
func UpdateVideoType(c *gin.Context) {
	db := config.GetDB()

	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "ID参数缺失",
		})
		return
	}

	var updateData struct {
		UnifiedName string `json:"unified_name"`
		Sort        int    `json:"sort"`
		IsActive    bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数解析失败",
		})
		return
	}

	result := db.Model(&models.VideoType{}).Where("id = ?", id).Updates(map[string]interface{}{
		"unified_name": updateData.UnifiedName,
		"sort":         updateData.Sort,
		"is_active":    updateData.IsActive,
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "更新成功",
		"data": gin.H{
			"affected": result.RowsAffected,
		},
	})
}

// SyncVideoTypes 同步分类信息（从videos表中提取所有分类）
func SyncVideoTypes(c *gin.Context) {
	db := config.GetDB()

	// 从videos表中获取所有唯一的分类
	var videoTypes []struct {
		TypeID     int
		TypeName   string
		SourceKey  string
		SourceName string
	}

	db.Model(&models.Video{}).
		Select("DISTINCT type_id, type_name, source_key, source_name").
		Where("type_id != 0 AND type_name != ''").
		Scan(&videoTypes)

	var created, updated int
	for _, vt := range videoTypes {
		var existingType models.VideoType

		// 检查是否已存在
		result := db.Where("type_id = ? AND source_key = ?", vt.TypeID, vt.SourceKey).
			First(&existingType)

		if result.Error != nil {
			// 不存在，创建新记录
			newType := models.VideoType{
				TypeID:     vt.TypeID,
				TypeName:   vt.TypeName,
				SourceKey:  vt.SourceKey,
				SourceName: vt.SourceName,
			}
			if err := db.Create(&newType).Error; err == nil {
				created++
			}
		} else {
			// 已存在，更新名称
			if existingType.TypeName != vt.TypeName {
				db.Model(&existingType).Update("type_name", vt.TypeName)
				updated++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "同步完成",
		"data": gin.H{
			"created": created,
			"updated": updated,
			"total":   len(videoTypes),
		},
	})
}

// GetUnifiedTypes 获取统一分类列表（用于跨源分类映射）
func GetUnifiedTypes(c *gin.Context) {
	db := config.GetDB()

	var unifiedTypes []struct {
		UnifiedName string `json:"unified_name"`
		Count       int64  `json:"count"`
		SourceKeys  string `json:"source_keys"`
	}

	db.Model(&models.VideoType{}).
		Select("unified_name, COUNT(*) as count, GROUP_CONCAT(DISTINCT source_key) as source_keys").
		Where("unified_name != ''").
		Group("unified_name").
		Order("count DESC").
		Scan(&unifiedTypes)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": unifiedTypes,
	})
}
