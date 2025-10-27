package models

import (
	"time"
)

// Video 视频模型
type Video struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 基本信息
	VodID      int    `gorm:"index" json:"vod_id"`
	VodName    string `gorm:"index;size:500" json:"vod_name"`
	VodEn      string `gorm:"size:500" json:"vod_en"`
	VodLetter  string `gorm:"size:10;index" json:"vod_letter"`
	VodPic     string `gorm:"size:1000" json:"vod_pic"`
	VodRemarks string `gorm:"size:200" json:"vod_remarks"`

	// 分类信息（原始）
	TypeID   int    `gorm:"index" json:"type_id"`            // 源站分类ID
	TypeID1  int    `gorm:"index" json:"type_id_1"`          // 源站父分类ID
	TypeName string `gorm:"size:100;index" json:"type_name"` // 源站分类名称
	VodClass string `gorm:"size:500" json:"vod_class"`       // 标签/类别

	// 分类信息（标准化）
	StandardCategoryID      int    `gorm:"index" json:"standard_category_id"`           // 标准一级分类ID
	StandardCategoryName    string `gorm:"size:50;index" json:"standard_category_name"` // 标准一级分类名称
	StandardSubCategoryID   *int   `gorm:"index" json:"standard_sub_category_id"`       // 标准二级分类ID
	StandardSubCategoryName string `gorm:"size:50" json:"standard_sub_category_name"`   // 标准二级分类名称

	// 旧字段（兼容）
	VideoTypeID uint       `gorm:"index" json:"video_type_id"` // 关联到video_types表
	VideoType   *VideoType `gorm:"foreignKey:VideoTypeID" json:"video_type,omitempty"`

	// 详细信息
	VodActor    string `gorm:"size:1000" json:"vod_actor"`
	VodDirector string `gorm:"size:500" json:"vod_director"`
	VodWriter   string `gorm:"size:500" json:"vod_writer"`
	VodBlurb    string `gorm:"size:1000" json:"vod_blurb"`
	VodContent  string `gorm:"type:text" json:"vod_content"`
	VodArea     string `gorm:"size:100;index" json:"vod_area"`
	VodLang     string `gorm:"size:100;index" json:"vod_lang"`
	VodYear     string `gorm:"size:50;index" json:"vod_year"`

	// 播放信息
	VodPlayFrom   string `gorm:"size:500" json:"vod_play_from"`
	VodPlayServer string `gorm:"size:500" json:"vod_play_server"`
	VodPlayNote   string `gorm:"size:500" json:"vod_play_note"`
	VodPlayURL    string `gorm:"type:text" json:"vod_play_url"`

	// 下载信息
	VodDownFrom   string `gorm:"size:500" json:"vod_down_from"`
	VodDownServer string `gorm:"size:500" json:"vod_down_server"`
	VodDownNote   string `gorm:"size:500" json:"vod_down_note"`
	VodDownURL    string `gorm:"type:text" json:"vod_down_url"`

	// 状态信息
	VodSerial   string `gorm:"size:100" json:"vod_serial"`
	VodState    string `gorm:"size:100" json:"vod_state"`
	VodIsEnd    int    `gorm:"index" json:"vod_isend"`
	VodDuration string `gorm:"size:100" json:"vod_duration"`

	// 评分信息
	VodScore       string  `gorm:"size:10" json:"vod_score"`
	VodScoreAll    int     `json:"vod_score_all"`
	VodScoreNum    int     `json:"vod_score_num"`
	VodDoubanID    int     `gorm:"index" json:"vod_douban_id"`
	VodDoubanScore float64 `json:"vod_douban_score"`

	// 统计信息
	VodHits      int `json:"vod_hits"`
	VodHitsDay   int `json:"vod_hits_day"`
	VodHitsWeek  int `json:"vod_hits_week"`
	VodHitsMonth int `json:"vod_hits_month"`

	// 其他信息
	VodPubdate   string `gorm:"size:200" json:"vod_pubdate"`
	VodLevel     int    `json:"vod_level"`
	VodCopyright int    `json:"vod_copyright"`
	VodLock      int    `json:"vod_lock"`
	GroupID      int    `gorm:"index" json:"group_id"`

	// 来源信息
	SourceKey   string    `gorm:"size:50;index;not null" json:"source_key"`
	SourceName  string    `gorm:"size:200;not null" json:"source_name"`
	CollectedAt time.Time `gorm:"index" json:"collected_at"`
}

// TableName 指定表名
func (Video) TableName() string {
	return "videos"
}
