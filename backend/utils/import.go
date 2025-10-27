package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"vodcms/config"
	"vodcms/models"
	"gorm.io/gorm"
)

// CategoryMappingHelper 分类映射辅助结构
type CategoryMappingHelper struct {
	mappings map[string]map[int]CategoryMapResult
}

// CategoryMapResult 映射结果
type CategoryMapResult struct {
	StandardID      int
	StandardSubID   *int
	StandardName    string
	StandardSubName string
}

// LoadCategoryMappings 加载分类映射
func LoadCategoryMappings() (*CategoryMappingHelper, error) {
	file, err := os.ReadFile("category_mapping.json")
	if err != nil {
		return nil, fmt.Errorf("读取分类映射文件失败: %w", err)
	}

	var config struct {
		StandardCategories map[string]struct {
			ID            int               `json:"id"`
			Name          string            `json:"name"`
			Subcategories map[string]string `json:"subcategories"`
		} `json:"standard_categories"`
		SourceMappings map[string]struct {
			Mappings []struct {
				SourceTypeID  int  `json:"source_type_id"`
				StandardID    int  `json:"standard_id"`
				StandardSubID *int `json:"standard_sub_category_id"`
			} `json:"mappings"`
		} `json:"source_mappings"`
	}

	if err := json.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("解析分类映射文件失败: %w", err)
	}

	helper := &CategoryMappingHelper{
		mappings: make(map[string]map[int]CategoryMapResult),
	}

	// 构建映射表
	for sourceKey, sourceMapping := range config.SourceMappings {
		helper.mappings[sourceKey] = make(map[int]CategoryMapResult)
		for _, mapping := range sourceMapping.Mappings {
			result := CategoryMapResult{
				StandardID:    mapping.StandardID,
				StandardSubID: mapping.StandardSubID,
			}

			// 获取标准分类名称
			if stdCat, ok := config.StandardCategories[fmt.Sprintf("%d", mapping.StandardID)]; ok {
				result.StandardName = stdCat.Name

				// 获取子分类名称
				if mapping.StandardSubID != nil {
					if subName, ok := stdCat.Subcategories[fmt.Sprintf("%d", *mapping.StandardSubID)]; ok {
						result.StandardSubName = subName
					}
				}
			}

			helper.mappings[sourceKey][mapping.SourceTypeID] = result
		}
	}

	return helper, nil
}

// MapCategory 映射分类
func (h *CategoryMappingHelper) MapCategory(sourceKey string, sourceTypeID int) CategoryMapResult {
	// 默认值
	defaultResult := CategoryMapResult{
		StandardID:   99,
		StandardName: "其他",
	}

	if sourceMappings, ok := h.mappings[sourceKey]; ok {
		if result, ok := sourceMappings[sourceTypeID]; ok {
			return result
		}
	}

	return defaultResult
}

// ImportVideoFromJSON 从JSON文件导入视频到数据库
func ImportVideoFromJSON(sourceKey string) error {
	db := config.GetDB()

	// 加载分类映射
	categoryHelper, err := LoadCategoryMappings()
	if err != nil {
		fmt.Printf("⚠️ 加载分类映射失败: %v，将使用原始分类\n", err)
		categoryHelper = nil
	}

	// 读取JSON文件
	filename := fmt.Sprintf("%s_vod.json", sourceKey)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 解析JSON
	var fileData struct {
		SourceInfo struct {
			Name    string `json:"name"`
			Key     string `json:"key"`
			BaseURL string `json:"base_url"`
		} `json:"source_info"`
		CollectionInfo struct {
			CollectedAt    string `json:"collected_at"`
			CollectionMode string `json:"collection_mode"`
			TotalCount     int    `json:"total_count"`
		} `json:"collection_info"`
		Videos []map[string]interface{} `json:"videos"`
	}

	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	fmt.Printf("📥 开始导入 %s 的视频数据，共 %d 条\n", fileData.SourceInfo.Name, len(fileData.Videos))

	// 批量导入
	successCount := 0
	updateCount := 0
	errorCount := 0

	for _, videoData := range fileData.Videos {
		video := mapToVideo(videoData)

		// 🔥 使用数据库映射规则（优先）+ JSON配置（备用）
		standardID, standardSubID, standardName, standardSubName := 
			mapCategoryWithDB(db, video.SourceKey, video.TypeID, video.TypeName, categoryHelper)

		video.StandardCategoryID = standardID
		video.StandardCategoryName = standardName
		video.StandardSubCategoryID = standardSubID
		video.StandardSubCategoryName = standardSubName

		// 检查是否已存在（根据vod_id和source_key）
		var existingVideo models.Video
		result := db.Where("vod_id = ? AND source_key = ?", video.VodID, video.SourceKey).First(&existingVideo)

		if result.RowsAffected > 0 {
			// 更新现有记录
			video.ID = existingVideo.ID
			video.CreatedAt = existingVideo.CreatedAt
			if err := db.Save(&video).Error; err != nil {
				fmt.Printf("  ❌ 更新失败 (ID:%d): %v\n", video.VodID, err)
				errorCount++
			} else {
				updateCount++
			}
		} else {
			// 创建新记录
			if err := db.Create(&video).Error; err != nil {
				fmt.Printf("  ❌ 创建失败 (ID:%d): %v\n", video.VodID, err)
				errorCount++
			} else {
				successCount++
			}
		}
	}

	fmt.Printf("✅ 导入完成: 新增 %d 条，更新 %d 条，失败 %d 条\n", successCount, updateCount, errorCount)
	return nil
}

// mapToVideo 将map转换为Video模型
func mapToVideo(data map[string]interface{}) models.Video {
	video := models.Video{}

	// 基本信息
	video.VodID = getInt(data, "vod_id")
	video.VodName = getString(data, "vod_name")
	video.VodEn = getString(data, "vod_en")
	video.VodLetter = getString(data, "vod_letter")
	video.VodPic = getString(data, "vod_pic")
	video.VodRemarks = getString(data, "vod_remarks")

	// 分类信息
	video.TypeID = getInt(data, "type_id")
	video.TypeID1 = getInt(data, "type_id_1")
	video.TypeName = getString(data, "type_name")
	video.VodClass = getString(data, "vod_class")

	// 详细信息
	video.VodActor = getString(data, "vod_actor")
	video.VodDirector = getString(data, "vod_director")
	video.VodWriter = getString(data, "vod_writer")
	video.VodBlurb = getString(data, "vod_blurb")
	video.VodContent = getString(data, "vod_content")
	video.VodArea = getString(data, "vod_area")
	video.VodLang = getString(data, "vod_lang")
	video.VodYear = getString(data, "vod_year")

	// 播放信息
	video.VodPlayFrom = getString(data, "vod_play_from")
	video.VodPlayServer = getString(data, "vod_play_server")
	video.VodPlayNote = getString(data, "vod_play_note")
	video.VodPlayURL = getString(data, "vod_play_url")

	// 下载信息
	video.VodDownFrom = getString(data, "vod_down_from")
	video.VodDownServer = getString(data, "vod_down_server")
	video.VodDownNote = getString(data, "vod_down_note")
	video.VodDownURL = getString(data, "vod_down_url")

	// 状态信息
	video.VodSerial = getString(data, "vod_serial")
	video.VodState = getString(data, "vod_state")
	video.VodIsEnd = getInt(data, "vod_isend")
	video.VodDuration = getString(data, "vod_duration")

	// 评分信息
	video.VodScore = getString(data, "vod_score")
	video.VodScoreAll = getInt(data, "vod_score_all")
	video.VodScoreNum = getInt(data, "vod_score_num")
	video.VodDoubanID = getInt(data, "vod_douban_id")
	video.VodDoubanScore = getFloat(data, "vod_douban_score")

	// 统计信息
	video.VodHits = getInt(data, "vod_hits")
	video.VodHitsDay = getInt(data, "vod_hits_day")
	video.VodHitsWeek = getInt(data, "vod_hits_week")
	video.VodHitsMonth = getInt(data, "vod_hits_month")

	// 其他信息
	video.VodPubdate = getString(data, "vod_pubdate")
	video.VodLevel = getInt(data, "vod_level")
	video.VodCopyright = getInt(data, "vod_copyright")
	video.VodLock = getInt(data, "vod_lock")
	video.GroupID = getInt(data, "group_id")

	// 来源信息
	video.SourceKey = getString(data, "source_key")
	video.SourceName = getString(data, "source_name")

	if collectedAt := getString(data, "collected_at"); collectedAt != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", collectedAt); err == nil {
			video.CollectedAt = t
		} else {
			video.CollectedAt = time.Now()
		}
	} else {
		video.CollectedAt = time.Now()
	}

	return video
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok && v != nil {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok && v != nil {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case string:
			// 尝试解析字符串
			if val == "0.0" {
				return 0.0
			}
		}
	}
	return 0.0
}

// mapCategoryWithDB 使用数据库规则映射分类（增强版）
// 优先级：1. 数据库精确规则 2. JSON配置 3. 默认值
func mapCategoryWithDB(db *gorm.DB, sourceKey string, sourceTypeID int, sourceTypeName string, helper *CategoryMappingHelper) (int, *int, string, string) {
	// 默认值
	defaultStandardID := 99
	defaultStandardName := "其他"
	var defaultStandardSubID *int = nil
	defaultStandardSubName := ""

	// 1. 优先从数据库查找精确匹配规则
	var rule models.MappingRule
	err := db.Where("source_key = ? AND source_type_id = ? AND is_active = ?",
		sourceKey, sourceTypeID, true).
		Order("priority ASC").
		First(&rule).Error

	if err == nil {
		// 找到数据库规则
		standardName, standardSubName := getStandardCategoryNames(db, rule.StandardID, rule.StandardSubID)
		return rule.StandardID, rule.StandardSubID, standardName, standardSubName
	}

	// 2. 从JSON配置查找（向后兼容）
	if helper != nil {
		result := helper.MapCategory(sourceKey, sourceTypeID)
		if result.StandardID != 99 {
			return result.StandardID, result.StandardSubID, result.StandardName, result.StandardSubName
		}
	}

	// 3. 返回默认值
	return defaultStandardID, defaultStandardSubID, defaultStandardName, defaultStandardSubName
}

// getStandardCategoryNames 从category_mapping.json获取标准分类名称
func getStandardCategoryNames(db *gorm.DB, standardID int, standardSubID *int) (string, string) {
	file, err := os.ReadFile("category_mapping.json")
	if err != nil {
		return "", ""
	}

	var config struct {
		StandardCategories map[string]struct {
			ID            int               `json:"id"`
			Name          string            `json:"name"`
			Subcategories map[string]string `json:"subcategories"`
		} `json:"standard_categories"`
	}

	if err := json.Unmarshal(file, &config); err != nil {
		return "", ""
	}

	stdKey := fmt.Sprintf("%d", standardID)
	stdCat, ok := config.StandardCategories[stdKey]
	if !ok {
		return "", ""
	}

	standardName := stdCat.Name
	standardSubName := ""

	if standardSubID != nil {
		subKey := fmt.Sprintf("%d", *standardSubID)
		if subName, ok := stdCat.Subcategories[subKey]; ok {
			standardSubName = subName
		}
	}

	return standardName, standardSubName
}
