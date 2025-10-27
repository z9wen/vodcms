package handles

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"vodcms/config"
	"vodcms/models"
)

// GetSources 获取数据源列表
func GetSources(c *gin.Context) {
	db := config.GetDB()

	var sources []models.Source
	result := db.Find(&sources)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": sources,
	})
}

// CreateSource 创建数据源
func CreateSource(c *gin.Context) {
	db := config.GetDB()

	var source models.Source
	if err := c.ShouldBindJSON(&source); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "无效的请求数据",
		})
		return
	}

	result := db.Create(&source)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "创建成功",
		"data": source,
	})
}

// UpdateSource 更新数据源
func UpdateSource(c *gin.Context) {
	db := config.GetDB()

	var source models.Source
	if err := c.ShouldBindJSON(&source); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "无效的请求数据",
		})
		return
	}

	result := db.Save(&source)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "更新成功",
		"data": source,
	})
}

// DeleteSource 删除数据源
func DeleteSource(c *gin.Context) {
	db := config.GetDB()

	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "ID参数缺失",
		})
		return
	}

	result := db.Delete(&models.Source{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})
}
