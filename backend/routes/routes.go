package routes

import (
	"vodcms/config"
	"vodcms/handles"
	"vodcms/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine) {
	db := config.GetDB()

	// 创建处理器实例
	mappingAdminHandler := handles.NewMappingAdminHandler(db)
	sourceDiscoveryHandler := handles.NewSourceDiscoveryHandler(db)

	// ============ 公开API（无需认证）============
	public := r.Group("/api")
	{
		// 健康检查
		public.GET("/health", healthCheck)

		// 视频相关路由（只读）
		public.GET("/videos", handles.GetVideos)
		public.GET("/videos/detail", handles.GetVideoByID)
		public.GET("/videos/play", handles.GetVideoPlayURL) // 获取播放地址
		public.GET("/videos/stats", handles.GetVideoStats)

		// 分类查询（只读）
		public.GET("/video-types", handles.GetVideoTypes)
		public.GET("/video-types/stats", handles.GetVideoTypeStats)
		public.GET("/categories", handles.GetStandardCategories)

		// 数据源查询（只读）
		public.GET("/sources", handles.GetSources)
	}

	// ============ 管理员API（需要认证）============
	admin := r.Group("/api/admin")
	admin.Use(middleware.AdminAuth())
	{
		// 【数据源管理】
		admin.POST("/sources/create", handles.CreateSource)
		admin.PUT("/sources/update", handles.UpdateSource)
		admin.DELETE("/sources/delete", handles.DeleteSource)

		// 【数据源发现和映射】
		admin.POST("/source/discover", sourceDiscoveryHandler.DiscoverSourceCategories)
		admin.POST("/source/auto-map", sourceDiscoveryHandler.AutoApplySuggestedMappings)
		admin.POST("/source/quick-map", sourceDiscoveryHandler.QuickMapCategory)
		admin.POST("/source/batch-map", sourceDiscoveryHandler.BatchQuickMap)
		admin.GET("/source/:source_key/mapping-status", sourceDiscoveryHandler.GetSourceMappingStatus)

		// 【分类管理】
		admin.PUT("/video-types/update", handles.UpdateVideoType)
		admin.POST("/video-types/sync", handles.SyncVideoTypes)
		admin.GET("/video-types/unified", handles.GetUnifiedTypes)
		admin.GET("/category-mappings", handles.GetCategoryMappings)

		// 【映射规则管理】
		admin.GET("/unmapped-categories", mappingAdminHandler.GetUnmappedCategories)
		admin.GET("/unmapped-categories/review", mappingAdminHandler.ReviewUnmappedCategories)
		admin.POST("/unmapped-categories/batch-apply", mappingAdminHandler.BatchApplyUnmappedCategories)
		admin.POST("/category-mapping/apply", mappingAdminHandler.ApplyCategoryMapping)

		admin.GET("/mapping-rules", mappingAdminHandler.GetMappingRules)
		admin.GET("/mapping-rules/preview", mappingAdminHandler.PreviewMappingRules)
		admin.POST("/mapping-rules", mappingAdminHandler.AddMappingRule)
		admin.POST("/mapping-rules/batch-update", mappingAdminHandler.BatchUpdateMappingRules)
		admin.POST("/mapping-rules/batch-delete", mappingAdminHandler.BatchDeleteMappingRules)
		admin.DELETE("/mapping-rules/:id", mappingAdminHandler.DeleteMappingRule)

		admin.GET("/fuzzy-rules", mappingAdminHandler.GetFuzzyMatchRules)
		admin.POST("/fuzzy-rules", mappingAdminHandler.AddFuzzyMatchRule)
		admin.GET("/mapping-stats", mappingAdminHandler.GetMappingStats)

		// 【采集管理】
		admin.POST("/collect", handles.CollectVideos)
		admin.GET("/collection-logs", handles.GetCollectionLogs)
		admin.POST("/import", handles.ImportJSON)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Server is running",
	})
}
