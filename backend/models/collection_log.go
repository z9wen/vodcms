package models

import "time"

// CollectionLog 采集日志模型
type CollectionLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	SourceName   string    `gorm:"size:200;index" json:"source_name"`
	SourceKey    string    `gorm:"size:50;index" json:"source_key"`
	Mode         string    `gorm:"size:20;index" json:"mode"`
	TotalPages   int       `json:"total_pages"`
	TotalVideos  int       `json:"total_videos"`
	SuccessCount int       `json:"success_count"`
	ErrorCount   int       `json:"error_count"`
	Duration     string    `gorm:"size:100" json:"duration"`
	StartTime    time.Time `gorm:"index" json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Status       string    `gorm:"size:20;index" json:"status"` // success, failed, partial
}

// TableName 指定表名
func (CollectionLog) TableName() string {
	return "collection_logs"
}
