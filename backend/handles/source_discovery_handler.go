package handles

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"vodcms/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SourceDiscoveryHandler 资源站发现处理器
type SourceDiscoveryHandler struct {
	db *gorm.DB
}

// NewSourceDiscoveryHandler 创建资源站发现处理器
func NewSourceDiscoveryHandler(db *gorm.DB) *SourceDiscoveryHandler {
	return &SourceDiscoveryHandler{
		db: db,
	}
}

// CategoryPreview 分类预览
type CategoryPreview struct {
	TypeID           int    `json:"type_id"`
	TypeName         string `json:"type_name"`
	Count            int    `json:"count"`
	Mapped           bool   `json:"mapped"`             // 是否已映射
	MappedTo         string `json:"mapped_to"`          // 映射到的标准分类（格式：1-101）
	SuggestedID      *int   `json:"suggested_id"`       // AI建议的标准分类ID
	SuggestedSubID   *int   `json:"suggested_sub_id"`   // AI建议的子分类ID
	SuggestedName    string `json:"suggested_name"`     // 建议的分类名称
	SuggestedSubName string `json:"suggested_sub_name"` // 建议的子分类名称
	Confidence       string `json:"confidence"`         // 置信度: high/medium/low
}

// DiscoverSourceCategories 发现资源站的分类
// POST /api/source/discover
// Body: {"source_key": "newzy", "api_url": "http://xxx.com/api.php/provide/vod/"}
func (h *SourceDiscoveryHandler) DiscoverSourceCategories(c *gin.Context) {
	var req struct {
		SourceKey string `json:"source_key" binding:"required"`
		APIURL    string `json:"api_url" binding:"required"`
		PageSize  int    `json:"page_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if req.PageSize == 0 {
		req.PageSize = 100
	}

	// 获取第一页数据来分析分类
	url := fmt.Sprintf("%s?ac=list&pg=1", req.APIURL)
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "无法连接到资源站: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取响应失败: " + err.Error()})
		return
	}

	var apiResp struct {
		Class []struct {
			TypeID   int    `json:"type_id"`
			TypeName string `json:"type_name"`
		} `json:"class"`
		List []struct {
			TypeID int `json:"type_id"`
		} `json:"list"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析响应失败: " + err.Error()})
		return
	}

	// 统计每个分类的数量
	categoryMap := make(map[int]*CategoryPreview)
	for _, class := range apiResp.Class {
		categoryMap[class.TypeID] = &CategoryPreview{
			TypeID:   class.TypeID,
			TypeName: class.TypeName,
			Count:    0,
			Mapped:   false,
		}
	}

	for _, video := range apiResp.List {
		if cat, exists := categoryMap[video.TypeID]; exists {
			cat.Count++
		}
	}

	// 检查哪些分类已经有映射规则，并为未映射的提供智能建议
	for typeID, cat := range categoryMap {
		var rule models.MappingRule
		err := h.db.Where("source_key = ? AND source_type_id = ? AND is_active = ?",
			req.SourceKey, typeID, true).First(&rule).Error

		if err == nil {
			// 已有映射规则
			cat.Mapped = true
			cat.MappedTo = fmt.Sprintf("%d", rule.StandardID)
			if rule.StandardSubID != nil {
				cat.MappedTo += fmt.Sprintf("-%d", *rule.StandardSubID)
			}
		} else {
			// 未映射，提供智能建议
			cat.Mapped = false
			suggestion := h.suggestMapping(cat.TypeName)
			cat.SuggestedID = suggestion.StandardID
			cat.SuggestedSubID = suggestion.StandardSubID
			cat.SuggestedName = suggestion.StandardName
			cat.SuggestedSubName = suggestion.StandardSubName
			cat.Confidence = suggestion.Confidence
		}
	}

	// 转为列表
	categories := make([]CategoryPreview, 0, len(categoryMap))
	mappedCount := 0
	unmappedCount := 0
	for _, cat := range categoryMap {
		categories = append(categories, *cat)
		if cat.Mapped {
			mappedCount++
		} else {
			unmappedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"source_key":     req.SourceKey,
			"api_url":        req.APIURL,
			"categories":     categories,
			"total_types":    len(categories),
			"mapped_count":   mappedCount,
			"unmapped_count": unmappedCount,
		},
		"message": fmt.Sprintf("发现 %d 个分类，已映射 %d 个，未映射 %d 个", len(categories), mappedCount, unmappedCount),
	})
}

// QuickMapCategory 快速映射分类
// POST /api/source/quick-map
// Body: {"source_key": "newzy", "source_type_id": 1, "source_name": "电影", "standard_id": 1, "standard_sub_id": null}
func (h *SourceDiscoveryHandler) QuickMapCategory(c *gin.Context) {
	var req struct {
		SourceKey     string `json:"source_key" binding:"required"`
		SourceTypeID  int    `json:"source_type_id" binding:"required"`
		SourceName    string `json:"source_name" binding:"required"`
		StandardID    int    `json:"standard_id" binding:"required"`
		StandardSubID *int   `json:"standard_sub_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 创建映射规则
	rule := models.MappingRule{
		SourceKey:     req.SourceKey,
		SourceTypeID:  req.SourceTypeID,
		SourceName:    req.SourceName,
		StandardID:    req.StandardID,
		StandardSubID: req.StandardSubID,
		Priority:      100,
		MatchType:     "exact",
		IsActive:      true,
	}

	// 检查是否已存在
	var existing models.MappingRule
	err := h.db.Where("source_key = ? AND source_type_id = ?", req.SourceKey, req.SourceTypeID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		if err := h.db.Create(&rule).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建映射失败: " + err.Error()})
			return
		}
	} else {
		if err := h.db.Model(&existing).Updates(&rule).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新映射失败: " + err.Error()})
			return
		}
		rule = existing
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "映射创建成功",
		"data":    rule,
	})
}

// BatchQuickMap 批量快速映射
// POST /api/source/batch-map
// Body: {"source_key": "newzy", "mappings": [{"source_type_id": 1, "source_name": "电影", "standard_id": 1}, ...]}
func (h *SourceDiscoveryHandler) BatchQuickMap(c *gin.Context) {
	var req struct {
		SourceKey string `json:"source_key" binding:"required"`
		Mappings  []struct {
			SourceTypeID  int    `json:"source_type_id" binding:"required"`
			SourceName    string `json:"source_name" binding:"required"`
			StandardID    int    `json:"standard_id" binding:"required"`
			StandardSubID *int   `json:"standard_sub_id"`
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
		rule := models.MappingRule{
			SourceKey:     req.SourceKey,
			SourceTypeID:  mapping.SourceTypeID,
			SourceName:    mapping.SourceName,
			StandardID:    mapping.StandardID,
			StandardSubID: mapping.StandardSubID,
			Priority:      100,
			MatchType:     "exact",
			IsActive:      true,
		}

		var existing models.MappingRule
		err := h.db.Where("source_key = ? AND source_type_id = ?", req.SourceKey, mapping.SourceTypeID).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			if err := h.db.Create(&rule).Error; err != nil {
				failCount++
				errors = append(errors, fmt.Sprintf("类型%d创建失败: %s", mapping.SourceTypeID, err.Error()))
				continue
			}
		} else {
			if err := h.db.Model(&existing).Updates(&rule).Error; err != nil {
				failCount++
				errors = append(errors, fmt.Sprintf("类型%d更新失败: %s", mapping.SourceTypeID, err.Error()))
				continue
			}
		}
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"success_count": successCount,
			"fail_count":    failCount,
			"errors":        errors,
		},
		"message": fmt.Sprintf("成功映射 %d 个分类，失败 %d 个", successCount, failCount),
	})
}

// GetSourceMappingStatus 获取资源站映射状态
// GET /api/source/:source_key/mapping-status
func (h *SourceDiscoveryHandler) GetSourceMappingStatus(c *gin.Context) {
	sourceKey := c.Param("source_key")

	var rules []models.MappingRule
	if err := h.db.Where("source_key = ? AND is_active = ?", sourceKey, true).Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取映射状态失败: " + err.Error()})
		return
	}

	var unmappedCount int64
	h.db.Model(&models.UnmappedCategory{}).Where("source_key = ? AND status = ?", sourceKey, "pending").Count(&unmappedCount)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"source_key":     sourceKey,
			"mapped_count":   len(rules),
			"unmapped_count": unmappedCount,
			"rules":          rules,
		},
	})
}

// MappingSuggestion 映射建议结构
type MappingSuggestion struct {
	StandardID      *int
	StandardSubID   *int
	StandardName    string
	StandardSubName string
	Confidence      string // high/medium/low
}

// suggestMapping 智能映射建议
// 基于分类名称提供映射建议
func (h *SourceDiscoveryHandler) suggestMapping(typeName string) MappingSuggestion {
	typeName = strings.TrimSpace(typeName)
	lowerName := strings.ToLower(typeName)

	// 高置信度匹配规则（精确匹配）
	highConfidenceRules := map[string]MappingSuggestion{
		// 电影
		"电影":   {StandardID: intPtr(1), StandardSubID: nil, StandardName: "电影", Confidence: "high"},
		"电影片":  {StandardID: intPtr(1), StandardSubID: nil, StandardName: "电影", Confidence: "high"},
		"动作片":  {StandardID: intPtr(1), StandardSubID: intPtr(101), StandardName: "电影", StandardSubName: "动作片", Confidence: "high"},
		"喜剧片":  {StandardID: intPtr(1), StandardSubID: intPtr(102), StandardName: "电影", StandardSubName: "喜剧片", Confidence: "high"},
		"爱情片":  {StandardID: intPtr(1), StandardSubID: intPtr(103), StandardName: "电影", StandardSubName: "爱情片", Confidence: "high"},
		"科幻片":  {StandardID: intPtr(1), StandardSubID: intPtr(104), StandardName: "电影", StandardSubName: "科幻片", Confidence: "high"},
		"恐怖片":  {StandardID: intPtr(1), StandardSubID: intPtr(105), StandardName: "电影", StandardSubName: "恐怖片", Confidence: "high"},
		"剧情片":  {StandardID: intPtr(1), StandardSubID: intPtr(106), StandardName: "电影", StandardSubName: "剧情片", Confidence: "high"},
		"战争片":  {StandardID: intPtr(1), StandardSubID: intPtr(107), StandardName: "电影", StandardSubName: "战争片", Confidence: "high"},
		"悬疑片":  {StandardID: intPtr(1), StandardSubID: intPtr(108), StandardName: "电影", StandardSubName: "悬疑片", Confidence: "high"},
		"犯罪片":  {StandardID: intPtr(1), StandardSubID: intPtr(109), StandardName: "电影", StandardSubName: "犯罪片", Confidence: "high"},
		"奇幻片":  {StandardID: intPtr(1), StandardSubID: intPtr(110), StandardName: "电影", StandardSubName: "奇幻片", Confidence: "high"},
		"灾难片":  {StandardID: intPtr(1), StandardSubID: intPtr(111), StandardName: "电影", StandardSubName: "灾难片", Confidence: "high"},
		"伦理片":  {StandardID: intPtr(1), StandardSubID: intPtr(112), StandardName: "电影", StandardSubName: "伦理片", Confidence: "high"},
		"伦理":   {StandardID: intPtr(1), StandardSubID: intPtr(112), StandardName: "电影", StandardSubName: "伦理片", Confidence: "high"},
		"4k电影": {StandardID: intPtr(1), StandardSubID: intPtr(113), StandardName: "电影", StandardSubName: "4K电影", Confidence: "high"},

		// 电视剧
		"电视剧": {StandardID: intPtr(2), StandardSubID: nil, StandardName: "电视剧", Confidence: "high"},
		"连续剧": {StandardID: intPtr(2), StandardSubID: nil, StandardName: "电视剧", Confidence: "high"},
		"国产剧": {StandardID: intPtr(2), StandardSubID: intPtr(201), StandardName: "电视剧", StandardSubName: "国产剧", Confidence: "high"},
		"大陆剧": {StandardID: intPtr(2), StandardSubID: intPtr(201), StandardName: "电视剧", StandardSubName: "国产剧", Confidence: "high"},
		"内地剧": {StandardID: intPtr(2), StandardSubID: intPtr(201), StandardName: "电视剧", StandardSubName: "国产剧", Confidence: "high"},
		"港澳剧": {StandardID: intPtr(2), StandardSubID: intPtr(202), StandardName: "电视剧", StandardSubName: "港澳剧", Confidence: "high"},
		"香港剧": {StandardID: intPtr(2), StandardSubID: intPtr(202), StandardName: "电视剧", StandardSubName: "港澳剧", Confidence: "high"},
		"港剧":  {StandardID: intPtr(2), StandardSubID: intPtr(202), StandardName: "电视剧", StandardSubName: "港澳剧", Confidence: "high"},
		"台湾剧": {StandardID: intPtr(2), StandardSubID: intPtr(203), StandardName: "电视剧", StandardSubName: "台湾剧", Confidence: "high"},
		"台剧":  {StandardID: intPtr(2), StandardSubID: intPtr(203), StandardName: "电视剧", StandardSubName: "台湾剧", Confidence: "high"},
		"欧美剧": {StandardID: intPtr(2), StandardSubID: intPtr(204), StandardName: "电视剧", StandardSubName: "欧美剧", Confidence: "high"},
		"美剧":  {StandardID: intPtr(2), StandardSubID: intPtr(204), StandardName: "电视剧", StandardSubName: "欧美剧", Confidence: "high"},
		"韩剧":  {StandardID: intPtr(2), StandardSubID: intPtr(205), StandardName: "电视剧", StandardSubName: "韩剧", Confidence: "high"},
		"韩国剧": {StandardID: intPtr(2), StandardSubID: intPtr(205), StandardName: "电视剧", StandardSubName: "韩剧", Confidence: "high"},
		"日剧":  {StandardID: intPtr(2), StandardSubID: intPtr(206), StandardName: "电视剧", StandardSubName: "日剧", Confidence: "high"},
		"日本剧": {StandardID: intPtr(2), StandardSubID: intPtr(206), StandardName: "电视剧", StandardSubName: "日剧", Confidence: "high"},
		"泰剧":  {StandardID: intPtr(2), StandardSubID: intPtr(207), StandardName: "电视剧", StandardSubName: "泰剧", Confidence: "high"},
		"马泰剧": {StandardID: intPtr(2), StandardSubID: intPtr(207), StandardName: "电视剧", StandardSubName: "泰剧", Confidence: "high"},
		"海外剧": {StandardID: intPtr(2), StandardSubID: intPtr(208), StandardName: "电视剧", StandardSubName: "海外剧", Confidence: "high"},

		// 综艺
		"综艺":   {StandardID: intPtr(3), StandardSubID: nil, StandardName: "综艺", Confidence: "high"},
		"大陆综艺": {StandardID: intPtr(3), StandardSubID: intPtr(301), StandardName: "综艺", StandardSubName: "大陆综艺", Confidence: "high"},
		"港台综艺": {StandardID: intPtr(3), StandardSubID: intPtr(302), StandardName: "综艺", StandardSubName: "港台综艺", Confidence: "high"},
		"日韩综艺": {StandardID: intPtr(3), StandardSubID: intPtr(303), StandardName: "综艺", StandardSubName: "日韩综艺", Confidence: "high"},
		"欧美综艺": {StandardID: intPtr(3), StandardSubID: intPtr(304), StandardName: "综艺", StandardSubName: "欧美综艺", Confidence: "high"},

		// 动漫
		"动漫":   {StandardID: intPtr(4), StandardSubID: nil, StandardName: "动漫", Confidence: "high"},
		"动画":   {StandardID: intPtr(4), StandardSubID: nil, StandardName: "动漫", Confidence: "high"},
		"国产动漫": {StandardID: intPtr(4), StandardSubID: intPtr(401), StandardName: "动漫", StandardSubName: "国产动漫", Confidence: "high"},
		"中国动漫": {StandardID: intPtr(4), StandardSubID: intPtr(401), StandardName: "动漫", StandardSubName: "国产动漫", Confidence: "high"},
		"日韩动漫": {StandardID: intPtr(4), StandardSubID: intPtr(402), StandardName: "动漫", StandardSubName: "日韩动漫", Confidence: "high"},
		"日本动漫": {StandardID: intPtr(4), StandardSubID: intPtr(402), StandardName: "动漫", StandardSubName: "日韩动漫", Confidence: "high"},
		"欧美动漫": {StandardID: intPtr(4), StandardSubID: intPtr(403), StandardName: "动漫", StandardSubName: "欧美动漫", Confidence: "high"},
		"港台动漫": {StandardID: intPtr(4), StandardSubID: intPtr(404), StandardName: "动漫", StandardSubName: "港台动漫", Confidence: "high"},
		"动画片":  {StandardID: intPtr(4), StandardSubID: intPtr(405), StandardName: "动漫", StandardSubName: "动画片", Confidence: "high"},
		"动漫电影": {StandardID: intPtr(4), StandardSubID: nil, StandardName: "动漫", Confidence: "high"},

		// 纪录片
		"纪录片": {StandardID: intPtr(5), StandardSubID: nil, StandardName: "纪录片", Confidence: "high"},
		"记录片": {StandardID: intPtr(5), StandardSubID: nil, StandardName: "纪录片", Confidence: "high"},

		// 短剧
		"短剧":   {StandardID: intPtr(6), StandardSubID: nil, StandardName: "短剧", Confidence: "high"},
		"爽文短剧": {StandardID: intPtr(6), StandardSubID: intPtr(601), StandardName: "短剧", StandardSubName: "爽文短剧", Confidence: "high"},
		"女频恋爱": {StandardID: intPtr(6), StandardSubID: intPtr(602), StandardName: "短剧", StandardSubName: "女频恋爱", Confidence: "high"},
		"反转爽剧": {StandardID: intPtr(6), StandardSubID: intPtr(603), StandardName: "短剧", StandardSubName: "反转爽剧", Confidence: "high"},
		"古装仙侠": {StandardID: intPtr(6), StandardSubID: intPtr(604), StandardName: "短剧", StandardSubName: "古装仙侠", Confidence: "high"},
		"年代穿越": {StandardID: intPtr(6), StandardSubID: intPtr(605), StandardName: "短剧", StandardSubName: "年代穿越", Confidence: "high"},
		"脑洞悬疑": {StandardID: intPtr(6), StandardSubID: intPtr(606), StandardName: "短剧", StandardSubName: "脑洞悬疑", Confidence: "high"},
		"现代都市": {StandardID: intPtr(6), StandardSubID: intPtr(607), StandardName: "短剧", StandardSubName: "现代都市", Confidence: "high"},
	}

	// 精确匹配
	if suggestion, ok := highConfidenceRules[typeName]; ok {
		return suggestion
	}

	// 中等置信度匹配（包含关键词）
	mediumConfidenceRules := []struct {
		keywords   []string
		suggestion MappingSuggestion
	}{
		{[]string{"动作", "武侠", "功夫"}, MappingSuggestion{intPtr(1), intPtr(101), "电影", "动作片", "medium"}},
		{[]string{"喜剧", "搞笑", "comedy"}, MappingSuggestion{intPtr(1), intPtr(102), "电影", "喜剧片", "medium"}},
		{[]string{"爱情", "浪漫", "言情"}, MappingSuggestion{intPtr(1), intPtr(103), "电影", "爱情片", "medium"}},
		{[]string{"科幻", "魔幻", "sci-fi"}, MappingSuggestion{intPtr(1), intPtr(104), "电影", "科幻片", "medium"}},
		{[]string{"恐怖", "惊悚", "鬼片", "horror"}, MappingSuggestion{intPtr(1), intPtr(105), "电影", "恐怖片", "medium"}},
		{[]string{"剧情", "文艺", "drama"}, MappingSuggestion{intPtr(1), intPtr(106), "电影", "剧情片", "medium"}},
		{[]string{"战争", "军事"}, MappingSuggestion{intPtr(1), intPtr(107), "电影", "战争片", "medium"}},
		{[]string{"悬疑", "推理", "mystery"}, MappingSuggestion{intPtr(1), intPtr(108), "电影", "悬疑片", "medium"}},
		{[]string{"movie", "film"}, MappingSuggestion{intPtr(1), nil, "电影", "", "medium"}},

		{[]string{"国产", "大陆", "内地", "chinese"}, MappingSuggestion{intPtr(2), intPtr(201), "电视剧", "国产剧", "medium"}},
		{[]string{"香港", "hk", "tvb"}, MappingSuggestion{intPtr(2), intPtr(202), "电视剧", "港澳剧", "medium"}},
		{[]string{"韩国", "korea", "korean"}, MappingSuggestion{intPtr(2), intPtr(205), "电视剧", "韩剧", "medium"}},
		{[]string{"日本", "japan", "japanese"}, MappingSuggestion{intPtr(2), intPtr(206), "电视剧", "日剧", "medium"}},
		{[]string{"欧美", "美国", "英国", "american", "usa"}, MappingSuggestion{intPtr(2), intPtr(204), "电视剧", "欧美剧", "medium"}},
		{[]string{"tv", "series", "剧集"}, MappingSuggestion{intPtr(2), nil, "电视剧", "", "medium"}},

		{[]string{"综艺", "真人秀", "variety"}, MappingSuggestion{intPtr(3), nil, "综艺", "", "medium"}},

		{[]string{"动漫", "动画", "anime", "cartoon"}, MappingSuggestion{intPtr(4), nil, "动漫", "", "medium"}},

		{[]string{"纪录", "记录", "documentary"}, MappingSuggestion{intPtr(5), nil, "纪录片", "", "medium"}},

		{[]string{"短剧", "微电影", "微剧"}, MappingSuggestion{intPtr(6), nil, "短剧", "", "medium"}},
	}

	for _, rule := range mediumConfidenceRules {
		for _, keyword := range rule.keywords {
			if strings.Contains(lowerName, keyword) {
				return rule.suggestion
			}
		}
	}

	// 低置信度 - 默认分类
	return MappingSuggestion{
		StandardID:      intPtr(99),
		StandardSubID:   nil,
		StandardName:    "其他",
		StandardSubName: "",
		Confidence:      "low",
	}
}

// intPtr 辅助函数：创建int指针
func intPtr(i int) *int {
	return &i
}

// AutoApplySuggestedMappings 自动应用建议的映射
// POST /api/source/auto-map
// Body: {"source_key": "newzy", "api_url": "http://xxx.com/api.php/provide/vod/", "confidence_threshold": "medium"}
func (h *SourceDiscoveryHandler) AutoApplySuggestedMappings(c *gin.Context) {
	var req struct {
		SourceKey           string `json:"source_key" binding:"required"`
		APIURL              string `json:"api_url" binding:"required"`
		ConfidenceThreshold string `json:"confidence_threshold"` // high/medium/low，默认 medium
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if req.ConfidenceThreshold == "" {
		req.ConfidenceThreshold = "medium"
	}

	// 获取分类信息
	url := fmt.Sprintf("%s?ac=list&pg=1", req.APIURL)
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "无法连接到资源站: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取响应失败: " + err.Error()})
		return
	}

	var apiResp struct {
		Class []struct {
			TypeID   int    `json:"type_id"`
			TypeName string `json:"type_name"`
		} `json:"class"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析响应失败: " + err.Error()})
		return
	}

	createdCount := 0
	skippedCount := 0
	lowConfidenceCount := 0
	var createdRules []models.MappingRule

	for _, class := range apiResp.Class {
		// 检查是否已存在映射
		var existing models.MappingRule
		err := h.db.Where("source_key = ? AND source_type_id = ?", req.SourceKey, class.TypeID).First(&existing).Error

		if err == nil {
			// 已存在，跳过
			skippedCount++
			continue
		}

		// 获取映射建议
		suggestion := h.suggestMapping(class.TypeName)

		// 根据置信度阈值决定是否创建
		shouldCreate := false
		switch req.ConfidenceThreshold {
		case "high":
			shouldCreate = suggestion.Confidence == "high"
		case "medium":
			shouldCreate = suggestion.Confidence == "high" || suggestion.Confidence == "medium"
		case "low":
			shouldCreate = true
		}

		if !shouldCreate {
			lowConfidenceCount++
			continue
		}

		// 创建映射规则
		if suggestion.StandardID != nil {
			rule := models.MappingRule{
				SourceKey:     req.SourceKey,
				SourceTypeID:  class.TypeID,
				SourceName:    class.TypeName,
				StandardID:    *suggestion.StandardID,
				StandardSubID: suggestion.StandardSubID,
				Priority:      100,
				MatchType:     "exact",
				IsActive:      true,
			}

			if err := h.db.Create(&rule).Error; err != nil {
				fmt.Printf("创建映射失败: %v\n", err)
				continue
			}

			createdRules = append(createdRules, rule)
			createdCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"source_key":           req.SourceKey,
			"created_count":        createdCount,
			"skipped_count":        skippedCount,
			"low_confidence_count": lowConfidenceCount,
			"confidence_threshold": req.ConfidenceThreshold,
			"created_rules":        createdRules,
		},
		"message": fmt.Sprintf("自动映射完成：创建 %d 个，跳过 %d 个，低置信度 %d 个", createdCount, skippedCount, lowConfidenceCount),
	})
}
