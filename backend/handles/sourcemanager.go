package handles

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// 数据源配置
type Source struct {
	Name    string `json:"name"`     // 源名称
	BaseURL string `json:"base_url"` // API地址
	Key     string `json:"key"`      // 源标识 (用于文件名)
	Enabled bool   `json:"enabled"`  // 是否启用
}

// 源管理器
type SourceManager struct {
	sources    []Source
	configFile string
}

// 创建源管理器
func NewSourceManager(configFile string) *SourceManager {
	return &SourceManager{
		sources:    make([]Source, 0),
		configFile: configFile,
	}
}

// 预定义的默认数据源
func (sm *SourceManager) GetDefaultSources() []Source {
	return []Source{
		{
			Name:    "豪华资源",
			BaseURL: "https://hhzyapi.com/api.php/provide/vod/from/hhm3u8/at/json",
			Key:     "hhzy",
			Enabled: true,
		},
		{
			Name:    "光速资源",
			BaseURL: "https://api.guangsuapi.com/api.php/provide/vod/from/gsm3u8/at/json",
			Key:     "guangsuzy",
			Enabled: true,
		},
		{
			Name:    "索尼资源",
			BaseURL: "https://suoniapi.com/api.php/provide/vod/from/snm3u8/at/json",
			Key:     "snzy",
			Enabled: true,
		},
		{
			Name:    "红牛资源",
			BaseURL: "https://www.hongniuzy2.com/api.php/provide/vod/from/hnm3u8/at/json/",
			Key:     "hnzy",
			Enabled: true,
		},
		{
			Name:    "新浪资源",
			BaseURL: "https://api.xinlangapi.com/xinlangapi.php/provide/vod/from/xlm3u8/at/json",
			Key:     "xlzy",
			Enabled: true,
		},
	}
}

// 加载源配置
func (sm *SourceManager) LoadSources() error {
	// 如果配置文件不存在，使用默认配置
	if _, err := os.Stat(sm.configFile); os.IsNotExist(err) {
		sm.sources = sm.GetDefaultSources()
		return sm.SaveSources() // 保存默认配置
	}

	data, err := ioutil.ReadFile(sm.configFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config struct {
		Sources []Source `json:"sources"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	sm.sources = config.Sources
	return nil
}

// 保存源配置
func (sm *SourceManager) SaveSources() error {
	config := struct {
		UpdatedAt string   `json:"updated_at"`
		Sources   []Source `json:"sources"`
	}{
		UpdatedAt: getCurrentTime(),
		Sources:   sm.sources,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON编码失败: %w", err)
	}

	return ioutil.WriteFile(sm.configFile, data, 0644)
}

// 获取所有启用的源
func (sm *SourceManager) GetEnabledSources() []Source {
	enabled := make([]Source, 0)
	for _, source := range sm.sources {
		if source.Enabled {
			enabled = append(enabled, source)
		}
	}
	return enabled
}

// 获取所有源
func (sm *SourceManager) GetAllSources() []Source {
	return sm.sources
}

// 添加新源
func (sm *SourceManager) AddSource(source Source) error {
	// 检查Key是否已存在
	for _, existing := range sm.sources {
		if existing.Key == source.Key {
			return fmt.Errorf("源标识 '%s' 已存在", source.Key)
		}
	}

	sm.sources = append(sm.sources, source)
	return sm.SaveSources()
}

// 更新源状态
func (sm *SourceManager) UpdateSourceStatus(key string, enabled bool) error {
	for i, source := range sm.sources {
		if source.Key == key {
			sm.sources[i].Enabled = enabled
			return sm.SaveSources()
		}
	}
	return fmt.Errorf("未找到源: %s", key)
}

// 删除源
func (sm *SourceManager) RemoveSource(key string) error {
	for i, source := range sm.sources {
		if source.Key == key {
			sm.sources = append(sm.sources[:i], sm.sources[i+1:]...)
			return sm.SaveSources()
		}
	}
	return fmt.Errorf("未找到源: %s", key)
}

// 显示源配置
func (sm *SourceManager) PrintSources() {
	fmt.Println("\n=== 数据源配置 ===")
	if len(sm.sources) == 0 {
		fmt.Println("没有配置任何数据源")
		return
	}

	for i, source := range sm.sources {
		status := "❌ 禁用"
		if source.Enabled {
			status = "✅ 启用"
		}
		fmt.Printf("%d. %s %s\n", i+1, source.Name, status)
		fmt.Printf("   Key: %s\n", source.Key)
		fmt.Printf("   URL: %s\n", source.BaseURL)
		fmt.Println()
	}
}

// 源管理交互界面
func (sm *SourceManager) ManageSources() {
	for {
		fmt.Println("\n=== 数据源管理 ===")
		fmt.Println("1. 查看所有源")
		fmt.Println("2. 启用/禁用源")
		fmt.Println("3. 添加新源")
		fmt.Println("4. 删除源")
		fmt.Println("5. 重置为默认配置")
		fmt.Println("0. 返回主菜单")

		var choice int
		fmt.Print("请选择操作: ")
		fmt.Scanf("%d", &choice)

		switch choice {
		case 0:
			return
		case 1:
			sm.PrintSources()
		case 2:
			sm.toggleSourceStatus()
		case 3:
			sm.addNewSource()
		case 4:
			sm.removeSource()
		case 5:
			sm.resetToDefault()
		default:
			fmt.Println("无效选择")
		}
	}
}

// 切换源状态
func (sm *SourceManager) toggleSourceStatus() {
	sm.PrintSources()
	fmt.Print("请输入要切换状态的源编号: ")
	var index int
	fmt.Scanf("%d", &index)

	if index < 1 || index > len(sm.sources) {
		fmt.Println("无效的源编号")
		return
	}

	source := &sm.sources[index-1]
	source.Enabled = !source.Enabled

	if err := sm.SaveSources(); err != nil {
		fmt.Printf("保存失败: %v\n", err)
	} else {
		status := "禁用"
		if source.Enabled {
			status = "启用"
		}
		fmt.Printf("✅ 已%s源: %s\n", status, source.Name)
	}
}

// 添加新源
func (sm *SourceManager) addNewSource() {
	var source Source

	fmt.Print("请输入源名称: ")
	fmt.Scanf("%s", &source.Name)

	fmt.Print("请输入源标识(用于文件名): ")
	fmt.Scanf("%s", &source.Key)

	fmt.Print("请输入API地址: ")
	fmt.Scanf("%s", &source.BaseURL)

	source.Enabled = true

	if err := sm.AddSource(source); err != nil {
		fmt.Printf("添加失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功添加源: %s\n", source.Name)
	}
}

// 删除源
func (sm *SourceManager) removeSource() {
	sm.PrintSources()
	fmt.Print("请输入要删除的源编号: ")
	var index int
	fmt.Scanf("%d", &index)

	if index < 1 || index > len(sm.sources) {
		fmt.Println("无效的源编号")
		return
	}

	source := sm.sources[index-1]
	if err := sm.RemoveSource(source.Key); err != nil {
		fmt.Printf("删除失败: %v\n", err)
	} else {
		fmt.Printf("✅ 已删除源: %s\n", source.Name)
	}
}

// 重置为默认配置
func (sm *SourceManager) resetToDefault() {
	fmt.Print("确认重置为默认配置? (y/N): ")
	var confirm string
	fmt.Scanf("%s", &confirm)

	if confirm == "y" || confirm == "Y" {
		sm.sources = sm.GetDefaultSources()
		if err := sm.SaveSources(); err != nil {
			fmt.Printf("重置失败: %v\n", err)
		} else {
			fmt.Println("✅ 已重置为默认配置")
		}
	}
}
