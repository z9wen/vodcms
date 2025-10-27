package handles

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"vodcms/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MappingAdminHandler 分类映射管理处理器
type MappingAdminHandler struct {
	db *gorm.DB
}

// NewMappingAdminHandler 创建分类映射管理处理器
func NewMappingAdminHandler(db *gorm.DB) *MappingAdminHandler {
	return &MappingAdminHandler{db: db}
}

// GetUnmappedCategories 获取未映射的分类
// GET /api/unmapped-categories?source_key=xxx&status=pending
func (h *MappingAdminHandler) GetUnmappedCategories(c *gin.Context) {
	sourceKey := c.Query("source_key")
	status := c.DefaultQuery("status", "pending")

	var categories []models.UnmappedCategory
	query := h.db.Model(&models.UnmappedCategory{})

	if sourceKey != "" {
		query = query.Where("source_key = ?", sourceKey)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("video_count DESC, last_seen_at DESC").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取未映射分类失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total":      len(categories),
			"categories": categories,
		},
	})
}

// ApplyCategoryMapping 应用分类映射
// POST /api/category-mapping/apply
func (h *MappingAdminHandler) ApplyCategoryMapping(c *gin.Context) {
	var req struct {
		UnmappedID    uint `json:"unmapped_id" binding:"required"`
		StandardID    int  `json:"standard_id" binding:"required"`
		StandardSubID *int `json:"standard_sub_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	var unmapped models.UnmappedCategory
	if err := h.db.First(&unmapped, req.UnmappedID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "未找到该分类"})
		return
	}

	// 创建映射规则
	rule := models.MappingRule{
		SourceKey:     unmapped.SourceKey,
		SourceTypeID:  unmapped.SourceTypeID,
		SourceName:    unmapped.SourceName,
		StandardID:    req.StandardID,
		StandardSubID: req.StandardSubID,
		Priority:      100,
		MatchType:     "exact",
		IsActive:      true,
	}

	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建规则失败: " + err.Error()})
		return
	}

	// 更新未映射分类状态
	updates := map[string]interface{}{
		"status":        "mapped",
		"mapped_id":     req.StandardID,
		"mapped_sub_id": req.StandardSubID,
	}
	if err := h.db.Model(&unmapped).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新状态失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "映射应用成功", "data": rule})
}

// AddMappingRule 添加映射规则
// POST /api/mapping-rules
func (h *MappingAdminHandler) AddMappingRule(c *gin.Context) {
	var rule models.MappingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if rule.Priority == 0 {
		rule.Priority = 100
	}
	if rule.MatchType == "" {
		rule.MatchType = "exact"
	}
	rule.IsActive = true

	// 检查是否已存在
	var existing models.MappingRule
	err := h.db.Where("source_key = ? AND source_type_id = ?", rule.SourceKey, rule.SourceTypeID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		if err := h.db.Create(&rule).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "添加规则失败: " + err.Error()})
			return
		}
	} else {
		if err := h.db.Model(&existing).Updates(&rule).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新规则失败: " + err.Error()})
			return
		}
		rule = existing
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "规则保存成功", "data": rule})
}

// GetMappingRules 获取映射规则列表
// GET /api/mapping-rules?source_key=xxx
func (h *MappingAdminHandler) GetMappingRules(c *gin.Context) {
	sourceKey := c.Query("source_key")

	var rules []models.MappingRule
	query := h.db.Model(&models.MappingRule{})

	if sourceKey != "" {
		query = query.Where("source_key = ?", sourceKey)
	}

	if err := query.Order("priority ASC, source_key ASC, source_type_id ASC").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取规则失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total": len(rules),
			"rules": rules,
		},
	})
}

// DeleteMappingRule 删除映射规则
// DELETE /api/mapping-rules/:id
func (h *MappingAdminHandler) DeleteMappingRule(c *gin.Context) {
	id := c.Param("id")
	ruleID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的规则ID"})
		return
	}

	var rule models.MappingRule
	if err := h.db.First(&rule, ruleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}

	if err := h.db.Model(&rule).Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除规则失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "规则已删除"})
}

// AddFuzzyMatchRule 添加模糊匹配规则
// POST /api/fuzzy-rules
func (h *MappingAdminHandler) AddFuzzyMatchRule(c *gin.Context) {
	var rule models.FuzzyMatchRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if rule.Priority == 0 {
		rule.Priority = 200
	}
	rule.IsActive = true

	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "添加模糊规则失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "模糊规则添加成功", "data": rule})
}

// GetFuzzyMatchRules 获取模糊匹配规则列表
// GET /api/fuzzy-rules
func (h *MappingAdminHandler) GetFuzzyMatchRules(c *gin.Context) {
	var rules []models.FuzzyMatchRule

	if err := h.db.Where("is_active = ?", true).Order("priority ASC").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取模糊规则失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total": len(rules),
			"rules": rules,
		},
	})
}

// GetMappingStats 获取映射统计信息
// GET /api/mapping-stats
func (h *MappingAdminHandler) GetMappingStats(c *gin.Context) {
	var unmappedCount, mappedCount, totalRules int64

	h.db.Model(&models.UnmappedCategory{}).Where("status = ?", "pending").Count(&unmappedCount)
	h.db.Model(&models.UnmappedCategory{}).Where("status = ?", "mapped").Count(&mappedCount)
	h.db.Model(&models.MappingRule{}).Where("is_active = ?", true).Count(&totalRules)

	stats := gin.H{
		"unmapped_pending": unmappedCount,
		"unmapped_mapped":  mappedCount,
		"total_rules":      totalRules,
		"updated_at":       time.Now().Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": stats})
}

// PreviewMappingRules 预览映射规则（支持筛选和排序）
// GET /api/admin/mapping-rules/preview?source_key=xxx&status=active
func (h *MappingAdminHandler) PreviewMappingRules(c *gin.Context) {
	sourceKey := c.Query("source_key")
	status := c.Query("status") // active/inactive/all

	type RulePreview struct {
		models.MappingRule
		VideoCount      int    `json:"video_count"`       // 使用该规则的视频数量
		StandardName    string `json:"standard_name"`     // 标准分类名称
		StandardSubName string `json:"standard_sub_name"` // 标准子分类名称
	}

	query := h.db.Model(&models.MappingRule{})

	if sourceKey != "" {
		query = query.Where("source_key = ?", sourceKey)
	}

	switch status {
	case "active":
		query = query.Where("is_active = ?", true)
	case "inactive":
		query = query.Where("is_active = ?", false)
	default: // "all"
		// 不过滤
	}

	var rules []models.MappingRule
	if err := query.Order("source_key ASC, priority ASC, source_type_id ASC").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取规则失败: " + err.Error()})
		return
	}

	// 增强规则信息
	var previews []RulePreview
	for _, rule := range rules {
		preview := RulePreview{MappingRule: rule}

		// 统计使用该规则的视频数量
		var count int64
		h.db.Model(&models.Video{}).Where(
			"source_key = ? AND type_id = ? AND standard_category_id = ?",
			rule.SourceKey, rule.SourceTypeID, rule.StandardID,
		).Count(&count)
		preview.VideoCount = int(count)

		// TODO: 获取标准分类名称（从category_mapping.json）
		// 这里简化处理，实际可以加载配置文件

		previews = append(previews, preview)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total": len(previews),
			"rules": previews,
		},
	})
}

// BatchUpdateMappingRules 批量更新映射规则状态
// POST /api/admin/mapping-rules/batch-update
func (h *MappingAdminHandler) BatchUpdateMappingRules(c *gin.Context) {
	var req struct {
		RuleIDs  []uint `json:"rule_ids" binding:"required"`
		IsActive *bool  `json:"is_active"` // 是否启用
		Priority *int   `json:"priority"`  // 优先级
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if len(req.RuleIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "规则ID列表不能为空"})
		return
	}

	updates := make(map[string]interface{})
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有需要更新的字段"})
		return
	}

	result := h.db.Model(&models.MappingRule{}).Where("id IN ?", req.RuleIDs).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败: " + result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "批量更新成功",
		"data": gin.H{
			"affected": result.RowsAffected,
		},
	})
}

// BatchDeleteMappingRules 批量删除（停用）映射规则
// POST /api/admin/mapping-rules/batch-delete
func (h *MappingAdminHandler) BatchDeleteMappingRules(c *gin.Context) {
	var req struct {
		RuleIDs []uint `json:"rule_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if len(req.RuleIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "规则ID列表不能为空"})
		return
	}

	result := h.db.Model(&models.MappingRule{}).Where("id IN ?", req.RuleIDs).Update("is_active", false)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败: " + result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "批量删除成功",
		"data": gin.H{
			"affected": result.RowsAffected,
		},
	})
}

// ReviewUnmappedCategories 审核未映射分类（带建议）
// GET /api/admin/unmapped-categories/review?source_key=xxx
func (h *MappingAdminHandler) ReviewUnmappedCategories(c *gin.Context) {
	sourceKey := c.Query("source_key")

	type UnmappedReview struct {
		models.UnmappedCategory
		SuggestedMapping string `json:"suggested_mapping"` // 建议的映射描述
	}

	query := h.db.Model(&models.UnmappedCategory{}).Where("status = ?", "pending")

	if sourceKey != "" {
		query = query.Where("source_key = ?", sourceKey)
	}

	var categories []models.UnmappedCategory
	if err := query.Order("video_count DESC, last_seen_at DESC").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取未映射分类失败: " + err.Error()})
		return
	}

	var reviews []UnmappedReview
	for _, cat := range categories {
		review := UnmappedReview{UnmappedCategory: cat}

		if cat.SuggestedID != nil {
			review.SuggestedMapping = "建议映射到分类 " + strconv.Itoa(*cat.SuggestedID)
			if cat.SuggestedSubID != nil {
				review.SuggestedMapping += "-" + strconv.Itoa(*cat.SuggestedSubID)
			}
		} else {
			review.SuggestedMapping = "需要手动指定"
		}

		reviews = append(reviews, review)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total":      len(reviews),
			"categories": reviews,
		},
	})
}

// BatchApplyUnmappedCategories 批量应用未映射分类
// POST /api/admin/unmapped-categories/batch-apply
func (h *MappingAdminHandler) BatchApplyUnmappedCategories(c *gin.Context) {
	var req struct {
		Mappings []struct {
			UnmappedID    uint `json:"unmapped_id" binding:"required"`
			StandardID    int  `json:"standard_id" binding:"required"`
			StandardSubID *int `json:"standard_sub_id"`
		} `json:"mappings" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	successCount := 0
	failCount := 0
	var errors []string

	for _, mapping := range req.Mappings {
		var unmapped models.UnmappedCategory
		if err := h.db.First(&unmapped, mapping.UnmappedID).Error; err != nil {
			failCount++
			errors = append(errors, fmt.Sprintf("未映射分类ID %d 不存在", mapping.UnmappedID))
			continue
		}

		// 创建映射规则
		rule := models.MappingRule{
			SourceKey:     unmapped.SourceKey,
			SourceTypeID:  unmapped.SourceTypeID,
			SourceName:    unmapped.SourceName,
			StandardID:    mapping.StandardID,
			StandardSubID: mapping.StandardSubID,
			Priority:      100,
			MatchType:     "exact",
			IsActive:      true,
		}

		if err := h.db.Create(&rule).Error; err != nil {
			failCount++
			errors = append(errors, fmt.Sprintf("创建规则失败: %s", err.Error()))
			continue
		}

		// 更新未映射分类状态
		updates := map[string]interface{}{
			"status":        "mapped",
			"mapped_id":     mapping.StandardID,
			"mapped_sub_id": mapping.StandardSubID,
		}
		if err := h.db.Model(&unmapped).Updates(updates).Error; err != nil {
			failCount++
			errors = append(errors, fmt.Sprintf("更新状态失败: %s", err.Error()))
			continue
		}

		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": fmt.Sprintf("批量应用完成：成功 %d 个，失败 %d 个", successCount, failCount),
		"data": gin.H{
			"success_count": successCount,
			"fail_count":    failCount,
			"errors":        errors,
		},
	})
}
