package handles

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// 采集模式
type CollectMode string

const (
	CollectAll   CollectMode = "all"   // 全部
	CollectToday CollectMode = "today" // 今天（24小时）
	CollectWeek  CollectMode = "week"  // 一周（168小时）
	CollectMonth CollectMode = "month" // 一个月（720小时）
)

// 采集结果统计
type CollectionStats struct {
	SourceName   string `json:"source_name"`
	SourceKey    string `json:"source_key"`
	TotalPages   int    `json:"total_pages"`
	TotalVideos  int    `json:"total_videos"`
	SuccessCount int    `json:"success_count"`
	ErrorCount   int    `json:"error_count"`
	Duration     string `json:"duration"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	FilePath     string `json:"file_path"`
}

// 苹果CMS API响应结构
type AppleCMSResponse struct {
	Code      int                      `json:"code"`
	Msg       string                   `json:"msg"`
	Page      interface{}              `json:"page"`
	PageCount interface{}              `json:"pagecount"`
	Limit     interface{}              `json:"limit"`
	Total     interface{}              `json:"total"`
	List      []map[string]interface{} `json:"list"`
	Class     []Category               `json:"class"`
}

type Category struct {
	TypeID   int    `json:"type_id"`
	TypePID  int    `json:"type_pid"`
	TypeName string `json:"type_name"`
}

// 采集器
type Collector struct {
	client *http.Client
}

// 创建采集器
func NewCollector() *Collector {
	return &Collector{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// 辅助函数：类型转换
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	}
	return ""
}

// 根据模式构建URL
func (c *Collector) buildURL(source Source, page int, mode CollectMode) string {
	baseURL := source.BaseURL

	switch mode {
	case CollectToday:
		return fmt.Sprintf("%s?ac=videolist&pg=%d&h=24", baseURL, page)
	case CollectWeek:
		return fmt.Sprintf("%s?ac=videolist&pg=%d&h=168", baseURL, page)
	case CollectMonth:
		return fmt.Sprintf("%s?ac=videolist&pg=%d&h=720", baseURL, page)
	default: // CollectAll
		return fmt.Sprintf("%s?ac=videolist&pg=%d", baseURL, page)
	}
}

// 获取数据
func (c *Collector) fetchData(url string) (*AppleCMSResponse, error) {
	fmt.Printf("  请求: %s\n", url)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result AppleCMSResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	if result.Code != 1 {
		return nil, fmt.Errorf("API返回错误: %s", result.Msg)
	}

	return &result, nil
}

// 采集单个源的数据
func (c *Collector) CollectSource(source Source, mode CollectMode, maxPages int) CollectionStats {
	startTime := time.Now()
	stats := CollectionStats{
		SourceName: source.Name,
		SourceKey:  source.Key,
		StartTime:  startTime.Format("2006-01-02 15:04:05"),
		FilePath:   fmt.Sprintf("%s_vod.json", source.Key),
	}

	fmt.Printf("\n=== 开始采集: %s ===\n", source.Name)

	// 获取第一页了解总数
	firstPageURL := c.buildURL(source, 1, mode)
	firstPage, err := c.fetchData(firstPageURL)
	if err != nil {
		stats.ErrorCount = 1
		stats.EndTime = time.Now().Format("2006-01-02 15:04:05")
		stats.Duration = time.Since(startTime).String()
		fmt.Printf("❌ %s 采集失败: %v\n", source.Name, err)
		return stats
	}

	pageCount := toInt(firstPage.PageCount)
	total := toInt(firstPage.Total)

	// 限制页数
	if maxPages > 0 && pageCount > maxPages {
		pageCount = maxPages
		fmt.Printf("⚠️ 限制采集页数为 %d 页 (总共 %d 页)\n", maxPages, toInt(firstPage.PageCount))
	}

	stats.TotalPages = pageCount
	stats.TotalVideos = total

	fmt.Printf("📊 将采集 %d 页，预计 %d 条记录\n", pageCount, total)

	if total == 0 {
		stats.EndTime = time.Now().Format("2006-01-02 15:04:05")
		stats.Duration = time.Since(startTime).String()
		fmt.Printf("⚠️ %s 没有找到符合条件的数据\n", source.Name)
		return stats
	}

	// 收集所有数据
	allData := make([]map[string]interface{}, 0)
	successCount := 0
	errorCount := 0

	// 处理第一页
	for _, videoMap := range firstPage.List {
		// 添加源信息
		videoMap["source_key"] = source.Key
		videoMap["source_name"] = source.Name
		videoMap["collected_at"] = getCurrentTime()
		allData = append(allData, videoMap)
		successCount++
	}

	// 处理剩余页面
	for page := 2; page <= pageCount; page++ {
		fmt.Printf("  采集第 %d/%d 页...\n", page, pageCount)

		pageURL := c.buildURL(source, page, mode)
		pageData, err := c.fetchData(pageURL)
		if err != nil {
			fmt.Printf("  ❌ 第 %d 页失败: %v\n", page, err)
			errorCount++
			continue
		}

		for _, videoMap := range pageData.List {
			videoMap["source_key"] = source.Key
			videoMap["source_name"] = source.Name
			videoMap["collected_at"] = getCurrentTime()
			allData = append(allData, videoMap)
			successCount++
		}

		time.Sleep(500 * time.Millisecond) // 避免请求过快
	}

	// 保存数据到文件
	if err := c.saveSourceData(source, allData, mode); err != nil {
		fmt.Printf("❌ 保存文件失败: %v\n", err)
		errorCount++
	} else {
		fmt.Printf("✅ 数据已保存到: %s\n", stats.FilePath)
	}

	stats.SuccessCount = successCount
	stats.ErrorCount = errorCount
	stats.EndTime = time.Now().Format("2006-01-02 15:04:05")
	stats.Duration = time.Since(startTime).String()

	fmt.Printf("✅ %s 采集完成: 成功 %d 条，失败 %d 条，耗时 %s\n",
		source.Name, successCount, errorCount, stats.Duration)

	return stats
}

// 保存单个源的数据
func (c *Collector) saveSourceData(source Source, data []map[string]interface{}, mode CollectMode) error {
	if len(data) == 0 {
		return fmt.Errorf("没有数据可保存")
	}

	// 创建文件数据结构
	fileData := map[string]interface{}{
		"source_info": map[string]interface{}{
			"name":     source.Name,
			"key":      source.Key,
			"base_url": source.BaseURL,
		},
		"collection_info": map[string]interface{}{
			"collected_at":    getCurrentTime(),
			"collection_mode": string(mode),
			"total_count":     len(data),
		},
		"videos": data,
	}

	// 转换为JSON
	jsonData, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON编码失败: %w", err)
	}

	// 保存文件
	filename := fmt.Sprintf("%s_vod.json", source.Key)
	err = ioutil.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	// 显示文件信息
	fileInfo, _ := os.Stat(filename)
	fmt.Printf("📁 文件大小: %.2f MB\n", float64(fileInfo.Size())/(1024*1024))

	return nil
}

// 批量采集多个源
func (c *Collector) CollectMultipleSources(sources []Source, mode CollectMode, maxPages int) []CollectionStats {
	if len(sources) == 0 {
		fmt.Println("❌ 没有启用的数据源")
		return nil
	}

	modeStr := map[CollectMode]string{
		CollectAll:   "全部数据",
		CollectToday: "今天更新",
		CollectWeek:  "本周更新",
		CollectMonth: "本月更新",
	}

	fmt.Printf("🚀 开始多源采集: %s\n", modeStr[mode])
	fmt.Printf("📋 启用的数据源: %d 个\n", len(sources))

	for _, source := range sources {
		fmt.Printf("  - %s (%s)\n", source.Name, source.Key)
	}

	// 依次采集每个源
	allStats := make([]CollectionStats, 0, len(sources))

	for _, source := range sources {
		stats := c.CollectSource(source, mode, maxPages)
		allStats = append(allStats, stats)
	}

	// 显示汇总统计
	c.printSummaryStats(allStats)

	// 保存采集报告
	c.saveCollectionReport(allStats, mode)

	return allStats
}

// 打印汇总统计
func (c *Collector) printSummaryStats(allStats []CollectionStats) {
	fmt.Printf("\n=== 采集汇总统计 ===\n")

	totalVideos := 0
	totalSuccess := 0
	totalErrors := 0

	for _, stat := range allStats {
		fmt.Printf("📊 %s:\n", stat.SourceName)
		fmt.Printf("   总页数: %d | 总视频: %d | 成功: %d | 失败: %d | 耗时: %s\n",
			stat.TotalPages, stat.TotalVideos, stat.SuccessCount, stat.ErrorCount, stat.Duration)
		fmt.Printf("   文件: %s\n", stat.FilePath)

		totalVideos += stat.TotalVideos
		totalSuccess += stat.SuccessCount
		totalErrors += stat.ErrorCount
	}

	fmt.Printf("\n📈 总计: 视频 %d 条 | 成功 %d 条 | 失败 %d 条\n",
		totalVideos, totalSuccess, totalErrors)
}

// 保存采集报告
func (c *Collector) saveCollectionReport(allStats []CollectionStats, mode CollectMode) {
	reportData := map[string]interface{}{
		"generated_at":     getCurrentTime(),
		"collection_mode":  string(mode),
		"total_sources":    len(allStats),
		"collection_stats": allStats,
	}

	jsonData, err := json.MarshalIndent(reportData, "", "  ")
	if err != nil {
		fmt.Printf("⚠️ 保存采集报告失败: %v\n", err)
		return
	}

	filename := fmt.Sprintf("collection_report_%s.json", getCurrentTimeForFile())
	err = ioutil.WriteFile(filename, jsonData, 0644)
	if err != nil {
		fmt.Printf("⚠️ 保存采集报告失败: %v\n", err)
		return
	}

	fmt.Printf("📋 采集报告已保存: %s\n", filename)
}

// 获取当前时间（用于显示）
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// 获取当前时间（用于文件名）
func getCurrentTimeForFile() string {
	return time.Now().Format("20060102_150405")
}
