package services

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"vodcms/models"

	"gorm.io/gorm"
)

// CategoryMapping 分类映射结构
type CategoryMapping struct {
	SourceTypeID  int    `json:"source_type_id"`
	SourceName    string `json:"source_name"`
	StandardID    int    `json:"standard_id"`
	StandardSubID *int   `json:"standard_sub_id"`
}

// StandardCategory 标准分类
type StandardCategory struct {
	ID            int               `json:"id"`
	Name          string            `json:"name"`
	Subcategories map[string]string `json:"subcategories"`
}

// CategoryMappingConfig 分类映射配置
type CategoryMappingConfig struct {
	UpdatedAt          string                           `json:"updated_at"`
	StandardCategories map[string]StandardCategory      `json:"standard_categories"`
	SourceMappings     map[string]SourceCategoryMapping `json:"source_mappings"`
}

// SourceCategoryMapping 资源站分类映射
type SourceCategoryMapping struct {
	Name     string            `json:"name"`
	Mappings []CategoryMapping `json:"mappings"`
}

// CategoryMappingService 分类映射服务
type CategoryMappingService struct {
	config     *CategoryMappingConfig
	configFile string
	db         *gorm.DB
	mu         sync.RWMutex
}

// NewCategoryMappingService 创建分类映射服务
func NewCategoryMappingService(configFile string, db *gorm.DB) *CategoryMappingService {
	service := &CategoryMappingService{
		configFile: configFile,
		db:         db,
	}
	service.LoadConfig()
	service.InitializeMappingRules()
	return service
}

// LoadConfig 加载配置文件
func (s *CategoryMappingService) LoadConfig() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.ReadFile(s.configFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config CategoryMappingConfig
	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	s.config = &config
	return nil
}

// MapCategory 映射分类
func (s *CategoryMappingService) MapCategory(sourceKey string, sourceTypeID int, sourceTypeName string) (standardID int, standardSubID *int, standardName string, standardSubName string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 默认值
	standardID = 99
	standardName = "其他"
	standardSubID = nil
	standardSubName = ""

	if s.config == nil {
		return
	}

	// 查找资源站映射
	sourceMapping, exists := s.config.SourceMappings[sourceKey]
	if !exists {
		return
	}

	// 查找匹配的映射
	for _, mapping := range sourceMapping.Mappings {
		if mapping.SourceTypeID == sourceTypeID {
			standardID = mapping.StandardID
			standardSubID = mapping.StandardSubID

			// 获取标准分类名称
			if stdCat, ok := s.config.StandardCategories[fmt.Sprintf("%d", standardID)]; ok {
				standardName = stdCat.Name

				// 获取子分类名称
				if standardSubID != nil {
					if subName, ok := stdCat.Subcategories[fmt.Sprintf("%d", *standardSubID)]; ok {
						standardSubName = subName
					}
				}
			}
			return
		}
	}

	// 如果没有找到映射，尝试通过名称匹配
	for _, mapping := range sourceMapping.Mappings {
		if mapping.SourceName == sourceTypeName {
			standardID = mapping.StandardID
			standardSubID = mapping.StandardSubID

			if stdCat, ok := s.config.StandardCategories[fmt.Sprintf("%d", standardID)]; ok {
				standardName = stdCat.Name
				if standardSubID != nil {
					if subName, ok := stdCat.Subcategories[fmt.Sprintf("%d", *standardSubID)]; ok {
						standardSubName = subName
					}
				}
			}
			return
		}
	}

	return
}

// GetStandardCategories 获取所有标准分类
func (s *CategoryMappingService) GetStandardCategories() map[string]StandardCategory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil
	}

	return s.config.StandardCategories
}

// GetSourceMappings 获取指定资源站的映射
func (s *CategoryMappingService) GetSourceMappings(sourceKey string) *SourceCategoryMapping {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil
	}

	if mapping, exists := s.config.SourceMappings[sourceKey]; exists {
		return &mapping
	}

	return nil
}

// GetCategoryStats 获取分类统计信息
func (s *CategoryMappingService) GetCategoryStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil
	}

	stats := map[string]interface{}{
		"standard_category_count": len(s.config.StandardCategories),
		"source_count":            len(s.config.SourceMappings),
	}

	// 统计每个资源站的映射数量
	sourceMappingCounts := make(map[string]int)
	for key, mapping := range s.config.SourceMappings {
		sourceMappingCounts[key] = len(mapping.Mappings)
	}
	stats["source_mapping_counts"] = sourceMappingCounts

	// 统计子分类数量
	subcategoryCount := 0
	for _, cat := range s.config.StandardCategories {
		subcategoryCount += len(cat.Subcategories)
	}
	stats["subcategory_count"] = subcategoryCount

	return stats
}

// InitializeMappingRules 初始化映射规则（从JSON配置导入到数据库）
func (s *CategoryMappingService) InitializeMappingRules() error {
	if s.db == nil {
		return nil // 如果没有数据库，仅使用JSON配置
	}

	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	if config == nil {
		return fmt.Errorf("配置文件未加载")
	}

	// 检查是否已经初始化
	var count int64
	s.db.Model(&models.MappingRule{}).Count(&count)
	if count > 0 {
		return nil // 已有规则，不重复初始化
	}

	// 从JSON配置导入映射规则到数据库
	for sourceKey, sourceMapping := range config.SourceMappings {
		for _, mapping := range sourceMapping.Mappings {
			rule := models.MappingRule{
				SourceKey:     sourceKey,
				SourceTypeID:  mapping.SourceTypeID,
				SourceName:    mapping.SourceName,
				StandardID:    mapping.StandardID,
				StandardSubID: mapping.StandardSubID,
				Priority:      100,
				MatchType:     "exact",
				IsActive:      true,
			}
			if err := s.db.Create(&rule).Error; err != nil {
				return fmt.Errorf("创建映射规则失败: %w", err)
			}
		}
	}

	// 创建一些默认的模糊匹配规则
	fuzzyRules := []models.FuzzyMatchRule{
		{Pattern: "动作|武侠|功夫", StandardID: 1, StandardSubID: intPtr(101), Priority: 200, IsActive: true},
		{Pattern: "喜剧|搞笑", StandardID: 1, StandardSubID: intPtr(102), Priority: 200, IsActive: true},
		{Pattern: "爱情|浪漫|言情", StandardID: 1, StandardSubID: intPtr(103), Priority: 200, IsActive: true},
		{Pattern: "科幻|魔幻", StandardID: 1, StandardSubID: intPtr(104), Priority: 200, IsActive: true},
		{Pattern: "恐怖|惊悚|鬼片", StandardID: 1, StandardSubID: intPtr(105), Priority: 200, IsActive: true},
		{Pattern: "国产|大陆|内地", StandardID: 2, StandardSubID: intPtr(201), Priority: 200, IsActive: true},
		{Pattern: "港剧|港片|香港", StandardID: 2, StandardSubID: intPtr(202), Priority: 200, IsActive: true},
		{Pattern: "韩剧|韩国", StandardID: 2, StandardSubID: intPtr(205), Priority: 200, IsActive: true},
		{Pattern: "日剧|日本", StandardID: 2, StandardSubID: intPtr(206), Priority: 200, IsActive: true},
		{Pattern: "欧美|美剧|英剧", StandardID: 2, StandardSubID: intPtr(204), Priority: 200, IsActive: true},
	}

	for _, rule := range fuzzyRules {
		if err := s.db.Create(&rule).Error; err != nil {
			return fmt.Errorf("创建模糊规则失败: %w", err)
		}
	}

	return nil
}

// MapCategoryEnhanced 增强版映射（支持数据库规则 + 自动学习）
func (s *CategoryMappingService) MapCategoryEnhanced(sourceKey string, sourceTypeID int, sourceTypeName string) (standardID int, standardSubID *int, standardName string, standardSubName string) {
	// 1. 优先从数据库查找精确匹配规则
	if s.db != nil {
		var rule models.MappingRule
		err := s.db.Where("source_key = ? AND source_type_id = ? AND is_active = ?",
			sourceKey, sourceTypeID, true).
			Order("priority ASC").
			First(&rule).Error

		if err == nil {
			standardID = rule.StandardID
			standardSubID = rule.StandardSubID
			standardName, standardSubName = s.getStandardCategoryNames(standardID, standardSubID)
			return
		}
	}

	// 2. 尝试从JSON配置查找（向后兼容）
	standardID, standardSubID, standardName, standardSubName = s.MapCategory(sourceKey, sourceTypeID, sourceTypeName)

	// 如果找到映射就返回
	if standardID != 99 {
		return
	}

	// 3. 尝试模糊匹配
	if s.db != nil && sourceTypeName != "" {
		var fuzzyRule models.FuzzyMatchRule
		err := s.db.Where("is_active = ?", true).
			Order("priority ASC").
			Find(&fuzzyRule).Error

		if err == nil {
			// 使用 LIKE 或正则匹配（这里简化为包含匹配）
			keywords := strings.Split(fuzzyRule.Pattern, "|")
			for _, keyword := range keywords {
				if strings.Contains(sourceTypeName, keyword) {
					standardID = fuzzyRule.StandardID
					standardSubID = fuzzyRule.StandardSubID
					standardName, standardSubName = s.getStandardCategoryNames(standardID, standardSubID)

					// 记录这次成功的模糊匹配，建议添加为精确规则
					s.recordUnmappedCategory(sourceKey, sourceTypeID, sourceTypeName, &standardID, standardSubID)
					return
				}
			}
		}
	}

	// 4. 未找到映射，记录为未映射分类
	s.recordUnmappedCategory(sourceKey, sourceTypeID, sourceTypeName, nil, nil)

	// 返回默认值
	standardID = 99
	standardName = "其他"
	return
}

// recordUnmappedCategory 记录未映射的分类
func (s *CategoryMappingService) recordUnmappedCategory(sourceKey string, sourceTypeID int, sourceTypeName string, suggestedID *int, suggestedSubID *int) {
	if s.db == nil {
		return
	}

	var unmapped models.UnmappedCategory
	result := s.db.Where("source_key = ? AND source_type_id = ?", sourceKey, sourceTypeID).First(&unmapped)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		unmapped = models.UnmappedCategory{
			SourceKey:      sourceKey,
			SourceTypeID:   sourceTypeID,
			SourceName:     sourceTypeName,
			VideoCount:     1,
			Status:         "pending",
			SuggestedID:    suggestedID,
			SuggestedSubID: suggestedSubID,
		}
		s.db.Create(&unmapped)
	} else {
		// 更新现有记录
		updates := map[string]interface{}{
			"last_seen_at": time.Now(),
			"video_count":  gorm.Expr("video_count + 1"),
		}
		if suggestedID != nil {
			updates["suggested_id"] = *suggestedID
			updates["suggested_sub_id"] = suggestedSubID
		}
		s.db.Model(&unmapped).Updates(updates)
	}
}

// getStandardCategoryNames 获取标准分类名称
func (s *CategoryMappingService) getStandardCategoryNames(standardID int, standardSubID *int) (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return "", ""
	}

	stdCat, ok := s.config.StandardCategories[fmt.Sprintf("%d", standardID)]
	if !ok {
		return "", ""
	}

	standardName := stdCat.Name
	standardSubName := ""

	if standardSubID != nil {
		if subName, ok := stdCat.Subcategories[fmt.Sprintf("%d", *standardSubID)]; ok {
			standardSubName = subName
		}
	}

	return standardName, standardSubName
}

// GetUnmappedCategories 获取未映射的分类列表
func (s *CategoryMappingService) GetUnmappedCategories(sourceKey string, status string) ([]models.UnmappedCategory, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var unmapped []models.UnmappedCategory
	query := s.db.Model(&models.UnmappedCategory{})

	if sourceKey != "" {
		query = query.Where("source_key = ?", sourceKey)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("video_count DESC, last_seen_at DESC").Find(&unmapped).Error
	return unmapped, err
}

// AddMappingRule 添加新的映射规则
func (s *CategoryMappingService) AddMappingRule(rule *models.MappingRule) error {
	if s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 检查是否已存在
	var existing models.MappingRule
	err := s.db.Where("source_key = ? AND source_type_id = ?", rule.SourceKey, rule.SourceTypeID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新规则
		return s.db.Create(rule).Error
	}

	// 更新现有规则
	return s.db.Model(&existing).Updates(rule).Error
}

// ApplyUnmappedCategoryMapping 应用未映射分类的映射
func (s *CategoryMappingService) ApplyUnmappedCategoryMapping(unmappedID uint, standardID int, standardSubID *int) error {
	if s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	var unmapped models.UnmappedCategory
	if err := s.db.First(&unmapped, unmappedID).Error; err != nil {
		return err
	}

	// 创建映射规则
	rule := models.MappingRule{
		SourceKey:     unmapped.SourceKey,
		SourceTypeID:  unmapped.SourceTypeID,
		SourceName:    unmapped.SourceName,
		StandardID:    standardID,
		StandardSubID: standardSubID,
		Priority:      100,
		MatchType:     "exact",
		IsActive:      true,
	}

	if err := s.AddMappingRule(&rule); err != nil {
		return err
	}

	// 更新未映射分类的状态
	updates := map[string]interface{}{
		"status":        "mapped",
		"mapped_id":     standardID,
		"mapped_sub_id": standardSubID,
	}
	return s.db.Model(&unmapped).Updates(updates).Error
}

// ExportMappingConfig 导出映射配置（用于备份或迁移）
func (s *CategoryMappingService) ExportMappingConfig() (*CategoryMappingConfig, error) {
	if s.db == nil {
		return s.config, nil
	}

	var rules []models.MappingRule
	if err := s.db.Where("is_active = ?", true).Find(&rules).Error; err != nil {
		return nil, err
	}

	// 构建配置
	config := &CategoryMappingConfig{
		UpdatedAt:          time.Now().Format("2006-01-02 15:04:05"),
		StandardCategories: s.config.StandardCategories,
		SourceMappings:     make(map[string]SourceCategoryMapping),
	}

	// 按源分组
	sourceMappings := make(map[string][]CategoryMapping)
	for _, rule := range rules {
		mapping := CategoryMapping{
			SourceTypeID:  rule.SourceTypeID,
			SourceName:    rule.SourceName,
			StandardID:    rule.StandardID,
			StandardSubID: rule.StandardSubID,
		}
		sourceMappings[rule.SourceKey] = append(sourceMappings[rule.SourceKey], mapping)
	}

	for sourceKey, mappings := range sourceMappings {
		config.SourceMappings[sourceKey] = SourceCategoryMapping{
			Name:     sourceKey,
			Mappings: mappings,
		}
	}

	return config, nil
}

// SaveConfigToFile 保存配置到文件
func (s *CategoryMappingService) SaveConfigToFile(filename string) error {
	config, err := s.ExportMappingConfig()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// Helper function
func intPtr(i int) *int {
	return &i
}
