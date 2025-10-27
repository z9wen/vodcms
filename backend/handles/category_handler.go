package handles

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// GetStandardCategories 获取标准分类列表
func GetStandardCategories(c *gin.Context) {
	file, err := os.ReadFile("category_mapping.json")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "读取分类配置失败",
		})
		return
	}

	var config struct {
		StandardCategories map[string]interface{} `json:"standard_categories"`
	}

	if err := json.Unmarshal(file, &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "解析分类配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": config.StandardCategories,
	})
}

// GetCategoryMappings 获取分类映射配置
func GetCategoryMappings(c *gin.Context) {
	sourceKey := c.Query("source_key")

	file, err := os.ReadFile("category_mapping.json")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "读取分类配置失败",
		})
		return
	}

	var config struct {
		SourceMappings map[string]interface{} `json:"source_mappings"`
	}

	if err := json.Unmarshal(file, &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "解析分类配置失败",
		})
		return
	}

	if sourceKey != "" {
		// 获取指定资源站的映射
		if mapping, ok := config.SourceMappings[sourceKey]; ok {
			c.JSON(http.StatusOK, gin.H{
				"code": 200,
				"msg":  "success",
				"data": mapping,
			})
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"code": 404,
				"msg":  "资源站不存在",
			})
		}
	} else {
		// 获取所有映射
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "success",
			"data": config.SourceMappings,
		})
	}
}
