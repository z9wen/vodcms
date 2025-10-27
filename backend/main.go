package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"vodcms/config"
	"vodcms/server"
)

func main() {
	// 解析命令行参数
	mode := flag.String("mode", "server", "运行模式: server (服务器模式) 或 cli (命令行模式)")
	port := flag.String("port", "8080", "服务器端口")
	flag.Parse()

	// 加载配置
	config.LoadConfig()

	// 初始化数据库
	if err := config.InitDatabase(); err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 苹果CMS多源采集系统 ===")

	switch *mode {
	case "server":
		// 服务器模式
		srv := server.NewServer(*port)
		if err := srv.Start(); err != nil {
			log.Fatalf("❌ 服务器启动失败: %v\n", err)
			os.Exit(1)
		}
	case "cli":
		// 命令行模式
		server.RunCLI()
	default:
		fmt.Printf("❌ 未知的运行模式: %s\n", *mode)
		fmt.Println("可用模式: server, cli")
		os.Exit(1)
	}
}
