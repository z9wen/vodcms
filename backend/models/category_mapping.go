package models

import "time"

// UnmappedCategory 未映射的分类记录
type UnmappedCategory struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SourceKey      string    `gorm:"index;size:50;not null" json:"source_key"` // 资源站标识
	SourceTypeID   int       `gorm:"index;not null" json:"source_type_id"`     // 资源站分类ID
	SourceName     string    `gorm:"size:100" json:"source_name"`              // 资源站分类名称
	FirstSeenAt    time.Time `gorm:"autoCreateTime" json:"first_seen_at"`      // 首次发现时间
	LastSeenAt     time.Time `gorm:"autoUpdateTime:milli" json:"last_seen_at"` // 最后一次遇到时间
	VideoCount     int       `gorm:"default:0" json:"video_count"`             // 该分类下的视频数量
	Status         string    `gorm:"size:20;default:'pending'" json:"status"`  // pending, mapped, ignored
	SuggestedID    *int      `json:"suggested_id"`                             // AI建议的标准分类ID
	SuggestedSubID *int      `json:"suggested_sub_id"`                         // AI建议的标准子分类ID
	MappedID       *int      `json:"mapped_id"`                                // 已映射的标准分类ID
	MappedSubID    *int      `json:"mapped_sub_id"`                            // 已映射的标准子分类ID
	Notes          string    `gorm:"type:text" json:"notes"`                   // 备注
}

// MappingRule 映射规则（数据库存储，可动态修改）
type MappingRule struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SourceKey     string    `gorm:"uniqueIndex:idx_mapping_source_type;size:50;not null" json:"source_key"`
	SourceTypeID  int       `gorm:"uniqueIndex:idx_mapping_source_type;not null" json:"source_type_id"`
	SourceName    string    `gorm:"size:100" json:"source_name"`
	StandardID    int       `gorm:"not null" json:"standard_id"`
	StandardSubID *int      `json:"standard_sub_id"`
	Priority      int       `gorm:"default:100" json:"priority"`               // 优先级，数字越小优先级越高
	MatchType     string    `gorm:"size:20;default:'exact'" json:"match_type"` // exact, fuzzy, pattern
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// FuzzyMatchRule 模糊匹配规则
type FuzzyMatchRule struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Pattern       string    `gorm:"size:100;not null" json:"pattern"` // 匹配模式（支持正则或关键词）
	Keywords      string    `gorm:"type:text" json:"keywords"`        // 关键词列表（JSON数组）
	StandardID    int       `gorm:"not null" json:"standard_id"`
	StandardSubID *int      `json:"standard_sub_id"`
	Priority      int       `gorm:"default:200" json:"priority"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (UnmappedCategory) TableName() string {
	return "unmapped_categories"
}

func (MappingRule) TableName() string {
	return "mapping_rules"
}

func (FuzzyMatchRule) TableName() string {
	return "fuzzy_match_rules"
}
