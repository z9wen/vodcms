package models

import (
	"time"
)

// VideoType 视频分类模型
type VideoType struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 分类ID（来自源站）
	TypeID int `gorm:"uniqueIndex:idx_source_type;not null" json:"type_id"`

	// 分类名称
	TypeName string `gorm:"size:100;not null;index" json:"type_name"`

	// 父分类ID（如果有层级关系）
	ParentID int `gorm:"index" json:"parent_id"`

	// 排序
	Sort int `gorm:"default:0" json:"sort"`

	// 来源信息（同一个TypeID在不同源可能有不同名称）
	SourceKey  string `gorm:"size:50;uniqueIndex:idx_source_type;not null" json:"source_key"`
	SourceName string `gorm:"size:200;not null" json:"source_name"`

	// 统一后的分类名（用于跨源合并）
	UnifiedName string `gorm:"size:100;index" json:"unified_name"`

	// 状态
	IsActive bool `gorm:"default:true" json:"is_active"`
}

// TableName 指定表名
func (VideoType) TableName() string {
	return "video_types"
}
