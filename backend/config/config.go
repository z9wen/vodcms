/*
Package config 配置管理包

项目结构说明：
================

项目目录结构：
/backend
├── main.go              # 程序入口，只负责启动应用
├── config/              # 配置相关
│   ├── config.go        # 应用配置
│   └── database.go      # 数据库连接和初始化
├── server/              # 服务器相关
│   └── server.go        # HTTP服务器和CLI启动逻辑
├── routes/              # 路由配置
│   └── routes.go        # API路由注册
├── handles/             # 业务逻辑处理层
│   ├── collector.go     # 视频采集器
│   ├── sourcemanager.go # 数据源管理器
│   ├── video_handler.go # 视频API处理
│   ├── source_handler.go # 数据源API处理
│   └── collection_handler.go # 采集API处理
├── services/            # 服务层（连接handle和model）
│   └── video_service.go # 视频服务（采集+数据库）
├── models/              # 数据库模型
│   ├── video.go         # 视频模型（根据JSON结构生成）
│   ├── source.go        # 数据源模型
│   └── collection_log.go # 采集日志模型
├── middleware/          # 中间件
│   └── cors.go          # CORS跨域中间件
└── utils/               # 工具函数

	├── import.go        # JSON导入工具
	└── response.go      # 统一响应格式

数据流向：
1. main.go -> 初始化配置和数据库 -> 启动server
2. server -> 注册routes -> handles处理请求
3. handles -> 调用services -> 操作models（数据库）
4. handles/collector.go -> 采集数据 -> 保存JSON文件
5. utils/import.go -> 读取JSON -> 保存到数据库

运行方式：
1. 服务器模式: ./vodcms --mode=server --port=8080
2. CLI模式:    ./vodcms --mode=cli
*/
package config

import "os"

type Config struct {
	ServerPort   string
	DatabasePath string
	SourceConfig string
}

var AppConfig *Config

// LoadConfig 加载配置
func LoadConfig() {
	AppConfig = &Config{
		ServerPort:   getEnv("PORT", "8080"),
		DatabasePath: getEnv("DB_PATH", "vodcms.db"),
		SourceConfig: getEnv("SOURCE_CONFIG", "sources_config.json"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
