package server

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"vodcms/handles"
	"vodcms/routes"
	"vodcms/services"
)

type Server struct {
	Port   string
	router *gin.Engine
}

// NewServer 创建服务器实例
func NewServer(port string) *Server {
	// 设置 Gin 模式 (release/debug)
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// CORS 中间件
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	return &Server{
		Port:   port,
		router: router,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 同步数据源到数据库
	videoService := services.NewVideoService()
	if err := videoService.SyncSourcesToDB(); err != nil {
		log.Printf("⚠️ 同步数据源失败: %v\n", err)
	}

	// 设置路由
	routes.SetupRoutes(s.router)

	fmt.Printf("服务器启动在端口: %s\n", s.Port)
	fmt.Printf("访问地址: http://localhost:%s\n", s.Port)

	if err := s.router.Run(":" + s.Port); err != nil {
		return fmt.Errorf("服务器启动失败: %w", err)
	}

	return nil
}

// RunCLI 运行命令行界面
func RunCLI() {
	videoService := services.NewVideoService()

	// 同步数据源
	if err := videoService.SyncSourcesToDB(); err != nil {
		log.Printf("同步数据源失败: %v\n", err)
	}

	for {
		showMainMenu()

		var choice int
		fmt.Print("请选择操作 (1-6): ")
		fmt.Scanf("%d", &choice)

		switch choice {
		case 1:
			collectWithMode(videoService, handles.CollectToday, "今天", 5)
		case 2:
			collectWithMode(videoService, handles.CollectWeek, "本周", 10)
		case 3:
			collectWithMode(videoService, handles.CollectMonth, "本月", 20)
		case 4:
			collectWithMode(videoService, handles.CollectAll, "全部", 0)
		case 5:
			manageSourcesMenu()
		case 6:
			fmt.Println("再见！")
			return
		default:
			fmt.Println("无效选择，请重试")
		}
	}
}

func showMainMenu() {
	fmt.Println("\n=== 主菜单 ===")
	fmt.Println("1. 📅 采集今天更新的视频 (24小时内)")
	fmt.Println("2. 📆 采集本周更新的视频 (168小时内)")
	fmt.Println("3. 📋 采集本月更新的视频 (720小时内)")
	fmt.Println("4. 🗂️  采集全部视频 (谨慎使用)")
	fmt.Println("5. ⚙️  管理数据源")
	fmt.Println("6. 🚪 退出程序")
}

func collectWithMode(videoService *services.VideoService, mode handles.CollectMode, modeName string, maxPages int) {
	modeDesc := map[int]string{
		5:  "限制 5 页",
		10: "限制 10 页",
		20: "限制 20 页",
		0:  "无限制",
	}

	fmt.Printf("\n将采集%s更新的视频，%s\n", modeName, modeDesc[maxPages])
	fmt.Print("确认开始采集? (y/N): ")

	var confirm string
	fmt.Scanf("%s", &confirm)

	if confirm == "y" || confirm == "Y" {
		fmt.Println("🚀 开始采集...")
		if err := videoService.CollectAndSave(mode, []string{}, maxPages); err != nil {
			fmt.Printf("❌ 采集失败: %v\n", err)
		} else {
			fmt.Println("✅ 采集完成！")
		}
	} else {
		fmt.Println("❌ 已取消采集")
	}
}

func manageSourcesMenu() {
	fmt.Println("\n=== 数据源管理 ===")
	fmt.Println("(功能开发中...)")
	// TODO: 实现数据源管理功能
}
