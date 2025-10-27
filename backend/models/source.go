package models

import "time"

// Source 数据源模型
type Source struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name    string `gorm:"size:200;not null" json:"name"`
	BaseURL string `gorm:"size:500;not null" json:"base_url"`
	Key     string `gorm:"size:50;uniqueIndex;not null" json:"key"`
	Enabled bool   `gorm:"default:true" json:"enabled"`
}

// TableName 指定表名
func (Source) TableName() string {
	return "sources"
}
