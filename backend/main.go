package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
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

// 修复：苹果CMS返回的字段类型问题（有些是string，有些是int）
type AppleCMSResponse struct {
	Code      int                      `json:"code"`
	Msg       string                   `json:"msg"`
	Page      interface{}              `json:"page"`      // 可能是string或int
	PageCount interface{}              `json:"pagecount"` // 可能是string或int
	Limit     interface{}              `json:"limit"`     // 可能是string或int
	Total     interface{}              `json:"total"`     // 可能是string或int
	List      []map[string]interface{} `json:"list"`      // 使用通用map处理复杂的字段
	Class     []Category               `json:"class"`
}

// 辅助函数：将interface{}转换为int
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

// 辅助函数：将interface{}转换为string
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

// 简化的视频结构（用于显示）
type SimpleVideo struct {
	VodID       int    `json:"vod_id"`
	VodName     string `json:"vod_name"`
	TypeName    string `json:"type_name"`
	VodRemarks  string `json:"vod_remarks"`
	VodTime     string `json:"vod_time"`
	VodPlayFrom string `json:"vod_play_from"`
	VodPic      string `json:"vod_pic"`
	VodPlayURL  string `json:"vod_play_url"`
	VodActor    string `json:"vod_actor"`
	VodDirector string `json:"vod_director"`
	VodContent  string `json:"vod_content"`
	VodArea     string `json:"vod_area"`
	VodYear     string `json:"vod_year"`
}

// 从map转换为SimpleVideo
func mapToSimpleVideo(m map[string]interface{}) SimpleVideo {
	return SimpleVideo{
		VodID:       toInt(m["vod_id"]),
		VodName:     toString(m["vod_name"]),
		TypeName:    toString(m["type_name"]),
		VodRemarks:  toString(m["vod_remarks"]),
		VodTime:     toString(m["vod_time"]),
		VodPlayFrom: toString(m["vod_play_from"]),
		VodPic:      toString(m["vod_pic"]),
		VodPlayURL:  toString(m["vod_play_url"]),
		VodActor:    toString(m["vod_actor"]),
		VodDirector: toString(m["vod_director"]),
		VodContent:  toString(m["vod_content"]),
		VodArea:     toString(m["vod_area"]),
		VodYear:     toString(m["vod_year"]),
	}
}

type Category struct {
	TypeID   int    `json:"type_id"`
	TypePID  int    `json:"type_pid"`
	TypeName string `json:"type_name"`
}

type PlayURL struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type AppleCMSCollector struct {
	client     *http.Client
	baseURL    string
	allRawData []map[string]interface{} // 保存所有原始数据
}

func NewAppleCMSCollector(baseURL string) *AppleCMSCollector {
	return &AppleCMSCollector{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    baseURL,
		allRawData: make([]map[string]interface{}, 0),
	}
}

// 根据模式获取列表页
func (c *AppleCMSCollector) FetchVideoListByMode(page int, mode CollectMode) (*AppleCMSResponse, error) {
	var url string

	switch mode {
	case CollectToday:
		url = fmt.Sprintf("%s?ac=videolist&pg=%d&h=24", c.baseURL, page)
	case CollectWeek:
		url = fmt.Sprintf("%s?ac=videolist&pg=%d&h=168", c.baseURL, page)
	case CollectMonth:
		url = fmt.Sprintf("%s?ac=videolist&pg=%d&h=720", c.baseURL, page)
	default: // CollectAll
		url = fmt.Sprintf("%s?ac=videolist&pg=%d", c.baseURL, page)
	}

	return c.fetchData(url)
}

// 获取详细信息（包含播放地址）
func (c *AppleCMSCollector) FetchVideoDetails(ids []int) ([]SimpleVideo, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("ids不能为空")
	}

	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.Itoa(id)
	}

	url := fmt.Sprintf("%s?ac=videolist&ids=%s", c.baseURL, strings.Join(idStrs, ","))

	resp, err := c.fetchData(url)
	if err != nil {
		return nil, err
	}

	// 转换为SimpleVideo
	videos := make([]SimpleVideo, len(resp.List))
	for i, videoMap := range resp.List {
		videos[i] = mapToSimpleVideo(videoMap)
	}

	return videos, nil
}

// 公共的数据获取方法
func (c *AppleCMSCollector) fetchData(url string) (*AppleCMSResponse, error) {
	fmt.Printf("请求URL: %s\n", url)

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

// 按模式采集
func (c *AppleCMSCollector) CollectByMode(mode CollectMode) error {
	modeStr := map[CollectMode]string{
		CollectAll:   "全部数据",
		CollectToday: "今天更新",
		CollectWeek:  "本周更新",
		CollectMonth: "本月更新",
	}

	fmt.Printf("开始采集: %s\n", modeStr[mode])

	// 1. 获取第一页了解总数
	firstPage, err := c.FetchVideoListByMode(1, mode)
	if err != nil {
		return fmt.Errorf("获取第一页失败: %w", err)
	}

	pageCount := toInt(firstPage.PageCount)
	total := toInt(firstPage.Total)

	fmt.Printf("找到 %d 页，共 %d 条记录\n", pageCount, total)

	if total == 0 {
		fmt.Println("没有找到符合条件的数据")
		return nil
	}

	// 2. 如果数据量不大，直接打印基本信息
	if total <= 50 {
		return c.printBasicInfo(firstPage, mode, pageCount)
	}

	// 3. 数据量大的话，采集详细信息
	return c.collectWithDetails(firstPage, mode, pageCount, total)
}

// 打印基本信息（适用于数据量小的情况）
func (c *AppleCMSCollector) printBasicInfo(firstPage *AppleCMSResponse, mode CollectMode, pageCount int) error {
	fmt.Println("\n=== 基本信息列表 ===")

	allVideos := make([]SimpleVideo, 0)

	// 处理第一页
	for _, videoMap := range firstPage.List {
		allVideos = append(allVideos, mapToSimpleVideo(videoMap))
		// 保存原始数据
		c.allRawData = append(c.allRawData, videoMap)
	}

	// 如果有多页，继续获取
	for page := 2; page <= pageCount; page++ {
		fmt.Printf("获取第 %d/%d 页...\n", page, pageCount)
		pageData, err := c.FetchVideoListByMode(page, mode)
		if err != nil {
			fmt.Printf("第 %d 页获取失败: %v\n", page, err)
			continue
		}

		for _, videoMap := range pageData.List {
			allVideos = append(allVideos, mapToSimpleVideo(videoMap))
			// 保存原始数据
			c.allRawData = append(c.allRawData, videoMap)
		}

		time.Sleep(500 * time.Millisecond)
	}

	// 打印所有视频基本信息
	for i, video := range allVideos {
		fmt.Printf("%d. [%s] %s (%s) - %s\n",
			i+1,
			video.TypeName,
			video.VodName,
			video.VodRemarks,
			video.VodTime)

		// 如果有播放地址，显示集数
		if video.VodPlayURL != "" {
			playURLs := c.ParsePlayURLs(video.VodPlayURL)
			fmt.Printf("   共 %d 集\n", len(playURLs))
		}
	}

	fmt.Printf("\n总计: %d 个视频\n", len(allVideos))

	// 保存到JSON文件
	if err := c.SaveToJSON("cmsdetails.json"); err != nil {
		fmt.Printf("保存JSON文件失败: %v\n", err)
	} else {
		fmt.Printf("✅ 数据已保存到 cmsdetails.json\n")
	}

	return nil
}

// 采集详细信息（适用于数据量大的情况）
func (c *AppleCMSCollector) collectWithDetails(firstPage *AppleCMSResponse, mode CollectMode, pageCount, total int) error {
	fmt.Println("\n=== 采集详细信息 ===")

	// 收集所有视频ID
	allVideoIDs := make([]int, 0, total)

	// 处理第一页
	for _, videoMap := range firstPage.List {
		allVideoIDs = append(allVideoIDs, toInt(videoMap["vod_id"]))
	}

	// 处理剩余页面
	for page := 2; page <= pageCount; page++ {
		fmt.Printf("获取第 %d/%d 页ID\n", page, pageCount)

		pageData, err := c.FetchVideoListByMode(page, mode)
		if err != nil {
			fmt.Printf("第 %d 页获取失败: %v\n", page, err)
			continue
		}

		for _, videoMap := range pageData.List {
			allVideoIDs = append(allVideoIDs, toInt(videoMap["vod_id"]))
		}

		time.Sleep(500 * time.Millisecond)
	}

	// 批量获取详细信息
	fmt.Printf("开始获取 %d 个视频的详细信息\n", len(allVideoIDs))

	batchSize := 5 // 减少批次大小，避免URL太长
	totalBatches := (len(allVideoIDs) + batchSize - 1) / batchSize

	for i := 0; i < len(allVideoIDs); i += batchSize {
		end := i + batchSize
		if end > len(allVideoIDs) {
			end = len(allVideoIDs)
		}

		batch := allVideoIDs[i:end]
		batchNum := i/batchSize + 1

		fmt.Printf("处理批次 %d/%d (ID: %d-%d)\n", batchNum, totalBatches, i+1, end)

		details, err := c.FetchVideoDetails(batch)
		if err != nil {
			fmt.Printf("批次 %d 获取详细信息失败: %v\n", batchNum, err)
			continue
		}

		// 获取原始详细数据并保存
		detailResp, err := c.FetchVideoDetailsRaw(batch)
		if err == nil {
			for _, videoMap := range detailResp.List {
				c.allRawData = append(c.allRawData, videoMap)
			}
		}

		// 打印详细信息
		for _, detail := range details {
			c.PrintVideoDetail(detail)
		}

		time.Sleep(1 * time.Second)
	}

	// 保存到JSON文件
	if err := c.SaveToJSON("cmsdetails.json"); err != nil {
		fmt.Printf("保存JSON文件失败: %v\n", err)
	} else {
		fmt.Printf("✅ 数据已保存到 cmsdetails.json\n")
	}

	return nil
}

// 打印视频详细信息
func (c *AppleCMSCollector) PrintVideoDetail(video SimpleVideo) {
	fmt.Printf("\n=== 视频详情 ===\n")
	fmt.Printf("ID: %d\n", video.VodID)
	fmt.Printf("名称: %s\n", video.VodName)
	fmt.Printf("分类: %s\n", video.TypeName)
	fmt.Printf("状态: %s\n", video.VodRemarks)
	fmt.Printf("年份: %s\n", video.VodYear)
	fmt.Printf("地区: %s\n", video.VodArea)
	fmt.Printf("更新时间: %s\n", video.VodTime)
	fmt.Printf("播放源: %s\n", video.VodPlayFrom)

	if video.VodPic != "" {
		fmt.Printf("封面图: %s\n", video.VodPic)
	}

	if video.VodActor != "" {
		fmt.Printf("演员: %s\n", video.VodActor)
	}

	if video.VodDirector != "" {
		fmt.Printf("导演: %s\n", video.VodDirector)
	}

	if video.VodContent != "" {
		// 限制简介长度
		content := video.VodContent
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		fmt.Printf("简介: %s\n", content)
	}

	if video.VodPlayURL != "" {
		fmt.Printf("播放地址:\n")
		playURLs := c.ParsePlayURLs(video.VodPlayURL)
		for i, playURL := range playURLs {
			if i < 5 { // 只显示前5集
				fmt.Printf("  %d. %s: %s\n", i+1, playURL.Name, playURL.URL)
			} else if i == 5 {
				fmt.Printf("  ... 共 %d 集\n", len(playURLs))
				break
			}
		}
	}

	fmt.Println(strings.Repeat("-", 50))
}

// 获取详细信息的原始数据（用于保存JSON）
func (c *AppleCMSCollector) FetchVideoDetailsRaw(ids []int) (*AppleCMSResponse, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("ids不能为空")
	}

	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.Itoa(id)
	}

	url := fmt.Sprintf("%s?ac=videolist&ids=%s", c.baseURL, strings.Join(idStrs, ","))
	return c.fetchData(url)
}

// 保存数据到JSON文件
func (c *AppleCMSCollector) SaveToJSON(filename string) error {
	if len(c.allRawData) == 0 {
		return fmt.Errorf("没有数据可保存")
	}

	// 创建包装结构，包含元数据
	saveData := map[string]interface{}{
		"collected_at": time.Now().Format("2006-01-02 15:04:05"),
		"source_url":   c.baseURL,
		"total_count":  len(c.allRawData),
		"videos":       c.allRawData,
	}

	// 转换为格式化的JSON
	jsonData, err := json.MarshalIndent(saveData, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON编码失败: %w", err)
	}

	// 写入文件
	err = ioutil.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	// 显示文件大小
	fileInfo, _ := os.Stat(filename)
	fmt.Printf("📁 文件大小: %.2f MB\n", float64(fileInfo.Size())/(1024*1024))

	return nil
}

// 从JSON文件加载数据
func (c *AppleCMSCollector) LoadFromJSON(filename string) error {
	jsonData, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	var loadData map[string]interface{}
	if err := json.Unmarshal(jsonData, &loadData); err != nil {
		return fmt.Errorf("JSON解码失败: %w", err)
	}

	// 提取视频数据
	if videos, ok := loadData["videos"].([]interface{}); ok {
		c.allRawData = make([]map[string]interface{}, len(videos))
		for i, video := range videos {
			if videoMap, ok := video.(map[string]interface{}); ok {
				c.allRawData[i] = videoMap
			}
		}
		fmt.Printf("✅ 从 %s 加载了 %d 条视频数据\n", filename, len(c.allRawData))
		return nil
	}

	return fmt.Errorf("JSON文件格式不正确")
}
func (c *AppleCMSCollector) ParsePlayURLs(playURLStr string) []PlayURL {
	var playURLs []PlayURL

	episodes := strings.Split(playURLStr, "#")
	for _, episode := range episodes {
		if episode == "" {
			continue
		}

		parts := strings.Split(episode, "$")
		if len(parts) >= 2 {
			playURLs = append(playURLs, PlayURL{
				Name: parts[0],
				URL:  parts[1],
			})
		}
	}

	return playURLs
}

// 主函数 - 测试不同模式
func main() {
	collector := NewAppleCMSCollector("https://hhzyapi.com/api.php/provide/vod/from/hhm3u8/at/json")

	fmt.Println("=== 苹果CMS数据采集器 ===")
	fmt.Println("选择操作:")
	fmt.Println("1. 采集今天更新的视频 (24小时内)")
	fmt.Println("2. 采集本周更新的视频 (168小时内)")
	fmt.Println("3. 采集本月更新的视频 (720小时内)")
	fmt.Println("4. 采集全部视频（谨慎使用，数据量很大）")
	fmt.Println("5. 从现有JSON文件加载数据")

	var choice int
	fmt.Print("请输入选择 (1-5): ")
	fmt.Scanf("%d", &choice)

	if choice == 5 {
		if err := collector.LoadFromJSON("cmsdetails.json"); err != nil {
			fmt.Printf("加载失败: %v\n", err)
		} else {
			fmt.Printf("已加载 %d 条数据\n", len(collector.allRawData))
			// 可以在这里添加数据分析功能
		}
		return
	}

	var mode CollectMode
	switch choice {
	case 1:
		mode = CollectToday
	case 2:
		mode = CollectWeek
	case 3:
		mode = CollectMonth
	case 4:
		mode = CollectAll
	default:
		fmt.Println("无效选择，默认使用今天更新")
		mode = CollectToday
	}

	if err := collector.CollectByMode(mode); err != nil {
		fmt.Printf("采集失败: %v\n", err)
	}
}
