package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 中间件
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// Logger 日志中间件
func Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: 实现日志记录
		next(w, r)
	}
}

// AdminAuth 管理员认证中间件
// 使用简单的Token认证，Token从环境变量或默认值获取
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取管理员Token（从环境变量或使用默认值）
		adminToken := os.Getenv("ADMIN_TOKEN")
		if adminToken == "" {
			adminToken = "vodcms_admin_2025" // 默认Token，建议在生产环境修改
		}

		// 从请求头获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "未提供认证信息",
			})
			c.Abort()
			return
		}

		// 支持 "Bearer token" 和 "token" 两种格式
		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		// 验证Token
		if token != adminToken {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "认证失败，无权限访问",
			})
			c.Abort()
			return
		}

		// 认证通过，继续处理
		c.Next()
	}
}

// OptionalAuth 可选认证中间件
// 如果提供了Token则验证，未提供则继续但标记为非管理员
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminToken := os.Getenv("ADMIN_TOKEN")
		if adminToken == "" {
			adminToken = "vodcms_admin_2025"
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			token = strings.TrimSpace(token)

			if token == adminToken {
				c.Set("is_admin", true)
			} else {
				c.Set("is_admin", false)
			}
		} else {
			c.Set("is_admin", false)
		}

		c.Next()
	}
}
