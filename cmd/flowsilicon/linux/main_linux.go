package main

import (
	"flowsilicon/internal/config"
	"flowsilicon/internal/logger"
	"flowsilicon/web"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"flowsilicon/internal/key"

	"github.com/getlantern/systray"
	"github.com/gin-gonic/gin"
)

var (
	// 全局变量，用于存储服务器端口
	serverPort int
	// 版本号
	Version = "1.3.6"
	// 系统托盘图标是否已隐藏
	iconHidden bool
	// 控制程序退出的通道
	quitChan chan struct{} = make(chan struct{})
	// 控制是否真正退出程序
	realQuit bool = false
	// 程序所在目录
	executableDir string
)

// getExecutableDir 获取可执行文件所在目录
func getExecutableDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法获取可执行文件路径: %v", err)
	}
	return filepath.Dir(execPath), nil
}

// getCurrentDir 获取当前工作目录
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "未知（获取失败）"
	}
	return dir
}

// getAbsolutePath 获取相对于可执行文件目录的绝对路径
func getAbsolutePath(relativePath string) string {
	return filepath.Join(executableDir, relativePath)
}

// openBrowser 打开默认浏览器访问指定URL
func openBrowser(url string) {
	var err error

	logger.Info("正在打开浏览器访问: %s", url)

	// Linux下使用xdg-open打开浏览器
	err = exec.Command("xdg-open", url).Start()

	if err != nil {
		logger.Error("打开浏览器失败: %v", err)
	}
}

// ensureConfigExists 确保配置文件存在，如果不存在则创建
func ensureConfigExists(configPath string) error {
	// 检查配置文件是否存在
	_, err := os.Stat(configPath)
	if err == nil {
		// 配置文件已存在
		return nil
	}

	if !os.IsNotExist(err) {
		// 发生了其他错误
		return fmt.Errorf("检查配置文件时出错: %v", err)
	}

	// 确保配置目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	return nil
}

// ensureDirectoriesExist 确保必要的目录结构存在
func ensureDirectoriesExist() error {
	// 需要确保存在的目录列表
	directories := []string{
		getAbsolutePath("config"),
		getAbsolutePath("data"),
		getAbsolutePath("logs"),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %v", dir, err)
		}
	}

	return nil
}

// 获取系统托盘图标
func getTrayIcon() []byte {
	// 首先尝试加载PNG格式图标（优先使用系统图标）
	iconPaths := []string{
		// 系统图标路径
		filepath.Join(os.Getenv("HOME"), ".local/share/icons/hicolor/128x128/apps/flowsilicon.png"),
		filepath.Join(os.Getenv("HOME"), ".local/share/icons/hicolor/64x64/apps/flowsilicon.png"),
		filepath.Join(os.Getenv("HOME"), ".local/share/icons/hicolor/48x48/apps/flowsilicon.png"),
		filepath.Join(os.Getenv("HOME"), ".local/share/icons/hicolor/32x32/apps/flowsilicon.png"),
		filepath.Join(os.Getenv("HOME"), ".local/share/icons/hicolor/24x24/apps/flowsilicon.png"),
		filepath.Join(os.Getenv("HOME"), ".local/share/icons/hicolor/16x16/apps/flowsilicon.png"),
		// 本地图标路径
		getAbsolutePath("icons/hicolor/128x128/apps/flowsilicon.png"),
		getAbsolutePath("icons/hicolor/64x64/apps/flowsilicon.png"),
		getAbsolutePath("icons/hicolor/48x48/apps/flowsilicon.png"),
		getAbsolutePath("icons/hicolor/32x32/apps/flowsilicon.png"),
		getAbsolutePath("icons/hicolor/24x24/apps/flowsilicon.png"),
		getAbsolutePath("icons/hicolor/16x16/apps/flowsilicon.png"),
		// ICO格式备选
		getAbsolutePath("web/static/favicon_16.ico"),
	}

	// 尝试加载图标
	for _, iconPath := range iconPaths {
		if _, err := os.Stat(iconPath); err == nil {
			// 图标文件存在，读取图标
			icon, err := os.ReadFile(iconPath)
			if err == nil {
				logger.Info("成功加载图标: %s", iconPath)
				return icon
			}
			logger.Error("读取图标文件失败: %v", err)
		}
	}

	// 如果所有路径都失败，返回空图标
	logger.Info("未找到任何有效的系统托盘图标，使用空图标")
	return make([]byte, 0)
}

// 检测当前Linux桌面环境
func detectDesktopEnvironment() string {
	// 检查环境变量
	desktopEnv := os.Getenv("XDG_CURRENT_DESKTOP")
	if desktopEnv != "" {
		return strings.ToLower(desktopEnv)
	}

	// 检查常见的环境变量
	for _, env := range []string{"GNOME_DESKTOP_SESSION_ID", "KDE_FULL_SESSION", "MATE_DESKTOP_SESSION_ID"} {
		if os.Getenv(env) != "" {
			return strings.ToLower(strings.Split(env, "_")[0])
		}
	}

	// 无法确定
	return "unknown"
}

// 系统托盘初始化
func onReady() {
	// 检测桌面环境
	desktopEnv := detectDesktopEnvironment()
	logger.Info("检测到Linux桌面环境: %s", desktopEnv)

	// 设置托盘图标和标题
	icon := getTrayIcon()
	if len(icon) > 0 {
		systray.SetIcon(icon)
	}

	// 获取配置
	cfg := config.GetConfig()
	// 设置图标隐藏状态
	iconHidden = cfg.App.HideIcon

	// 正常显示图标和标题
	systray.SetTitle("流动硅基")
	systray.SetTooltip("流动硅基 FlowSilicon v" + Version)

	// 添加菜单项
	mOpen := systray.AddMenuItem("打开界面", "打开Web界面")
	systray.AddSeparator()

	// 新增重载配置菜单项
	mReload := systray.AddMenuItem("重载配置", "重新加载配置文件")

	// 新增开机自动启动菜单项
	mAutoStart := systray.AddMenuItem("开机自动启动", "设置或取消开机自动启动")
	// 检查当前开机自动启动状态并设置选中状态
	if isAutoStartEnabled() {
		mAutoStart.Check()
	}

	// 新增最小化选项（在某些桌面环境下有用）
	mMinimize := systray.AddMenuItem("最小化到任务栏", "最小化应用程序到任务栏")

	// 新增隐藏图标菜单项
	mHideIcon := systray.AddMenuItem("隐藏图标", "隐藏系统托盘图标")

	// 如果设置为隐藏图标，则将其设置为极小图标
	if iconHidden {
		minimizeIcon()
		// 更新菜单项文本
		mHideIcon.SetTitle("显示图标")
		mHideIcon.SetTooltip("显示系统托盘图标")
	}

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出程序", "退出程序")

	// 处理菜单点击事件
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				// 打开Web界面
				openBrowser(fmt.Sprintf("http://localhost:%d", serverPort))
			case <-mReload.ClickedCh:
				// 重载配置
				reloadConfig()
				// 重载配置后检查图标状态
				cfg := config.GetConfig()
				if cfg.App.HideIcon && !iconHidden {
					// 配置为隐藏但图标当前显示，则隐藏图标
					iconHidden = true
					minimizeIcon()
					// 更新菜单项文本
					mHideIcon.SetTitle("显示图标")
					mHideIcon.SetTooltip("显示系统托盘图标")
				} else if !cfg.App.HideIcon && iconHidden {
					// 配置为显示但图标当前隐藏，则显示图标
					iconHidden = false
					restoreIcon()
					// 更新菜单项文本
					mHideIcon.SetTitle("隐藏图标")
					mHideIcon.SetTooltip("隐藏系统托盘图标")
				}
			case <-mAutoStart.ClickedCh:
				// 切换开机自动启动状态
				if isAutoStartEnabled() {
					disableAutoStart()
					mAutoStart.Uncheck()
					logger.Info("已禁用开机自动启动")
				} else {
					enableAutoStart()
					mAutoStart.Check()
					logger.Info("已启用开机自动启动")
				}
			case <-mMinimize.ClickedCh:
				// 最小化应用程序窗口（仅适用于桌面环境支持的情况）
				logger.Info("尝试最小化应用程序窗口")
				// 通知桌面环境最小化窗口
				notifyDesktopMinimize()
			case <-mHideIcon.ClickedCh:
				// 切换图标显示状态
				if iconHidden {
					// 当前隐藏，需要显示
					iconHidden = false
					restoreIcon()
					// 更新菜单项文本
					mHideIcon.SetTitle("隐藏图标")
					mHideIcon.SetTooltip("隐藏系统托盘图标")
					// 更新配置
					cfg := config.GetConfig()
					cfg.App.HideIcon = false
					config.UpdateConfig(cfg)
					logger.Info("已显示系统托盘图标")
				} else {
					// 当前显示，需要隐藏
					iconHidden = true
					// 更新配置
					cfg := config.GetConfig()
					cfg.App.HideIcon = true
					config.UpdateConfig(cfg)
					// 更新菜单项文本
					mHideIcon.SetTitle("显示图标")
					mHideIcon.SetTooltip("显示系统托盘图标")
					logger.Info("隐藏系统托盘图标")
					minimizeIcon()
				}
			case <-mQuit.ClickedCh:
				// 退出程序
				logger.Info("用户通过托盘菜单退出程序")
				realQuit = true // 设置真正退出标志
				systray.Quit()
				return
			}
		}
	}()
}

// notifyDesktopMinimize 通知桌面环境最小化窗口
func notifyDesktopMinimize() {
	// 尝试使用xdotool来控制窗口（需要安装xdotool）
	if _, err := exec.LookPath("xdotool"); err == nil {
		// 尝试找到并最小化窗口
		cmd := exec.Command("xdotool", "search", "--name", "流动硅基", "windowminimize")
		if err := cmd.Run(); err != nil {
			logger.Error("使用xdotool最小化窗口失败: %v", err)
		} else {
			logger.Info("成功最小化窗口")
		}
	} else {
		logger.Warn("未安装xdotool，无法控制窗口最小化。请安装: sudo apt-get install xdotool")
	}
}

// 系统托盘退出
func onExit() {
	// 如果是真正的退出请求，则退出程序
	if realQuit {
		// 保存API密钥
		config.SaveApiKeys()
		logger.Info("程序已退出")
		// 关闭退出通道，通知主程序退出
		close(quitChan)

		// 不立即调用os.Exit，让主程序进行正确的清理工作
		// os.Exit(0) 被移除，使用通道通知主程序
	} else {
		// 如果不是真正退出，只是重启systray（比如在隐藏/显示图标时）
		logger.Info("系统托盘重启中...")
		// 短暂等待以避免可能的冲突
		time.Sleep(100 * time.Millisecond)
		go systray.Run(onReady, onExit)
	}
}

func main() {
	// 获取可执行文件所在目录
	var err error
	executableDir, err = getExecutableDir()
	if err != nil {
		fmt.Printf("无法获取可执行文件目录: %v\n", err)
		os.Exit(1)
	}

	// Linux下的GUI模式判断，通过环境变量控制
	isGui := os.Getenv("FLOWSILICON_GUI") == "1"
	logger.SetGuiMode(isGui)

	// 初始化日志
	logger.InitLogger()

	// 记录启动模式
	if isGui {
		logger.Info("程序以GUI模式启动，日志仅写入文件")
	} else {
		logger.Info("程序以控制台模式启动，日志同时写入控制台和文件")
	}

	logger.Info("程序运行目录: %s", executableDir)
	// 添加更多路径信息用于调试
	logger.Info("数据文件绝对路径: %s", getAbsolutePath("data/api_keys.json"))
	logger.Info("配置文件绝对路径: %s", getAbsolutePath("config/config.yaml"))
	logger.Info("日志目录绝对路径: %s", getAbsolutePath("logs"))
	logger.Info("当前工作目录: %s", getCurrentDir())

	// 确保必要的目录结构存在
	if err := ensureDirectoriesExist(); err != nil {
		logger.Error("确保目录结构存在时出错: %v", err)
	} else {
		logger.Info("已确保必要的目录结构存在")
	}

	// 加载配置文件
	configPath := getAbsolutePath("config/config.yaml")
	if len(os.Args) > 1 && os.Args[1] == "--config" && len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	// 确保配置文件存在
	if err := ensureConfigExists(configPath); err != nil {
		logger.Error("确保配置文件存在时出错: %v", err)
	} else {
		logger.Info("已确保配置文件存在: %s", configPath)
	}

	// 尝试加载配置文件
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("加载配置文件失败: %v，将使用默认配置", err)
		// 使用默认配置
		cfg = config.GetConfig()
	} else {
		logger.Info("成功加载配置文件: %s", configPath)
		// 添加调试信息
		logger.Info("配置值 - AutoUpdateInterval: %d, StatsRefreshInterval: %d, RateRefreshInterval: %d",
			cfg.App.AutoUpdateInterval, cfg.App.StatsRefreshInterval, cfg.App.RateRefreshInterval)
	}

	// 设置日志文件大小，使用goroutine避免阻塞
	go func() {
		// 设置日志文件最大大小
		logMaxSize := cfg.Log.MaxSizeMB
		if logMaxSize <= 0 {
			logMaxSize = 10 // 默认10MB
		}
		logger.SetMaxLogSize(logMaxSize)

		// 手动触发一次日志清理
		time.Sleep(2 * time.Second) // 等待日志系统完全初始化
		logger.CleanLogsNow()
	}()

	// 设置数据文件路径
	config.SetApiKeysFile(getAbsolutePath("data/api_keys.json"))
	config.SetDailyFilePath(getAbsolutePath("data/daily.json"))

	// 初始化每日统计数据
	if err := config.InitDailyStats(); err != nil {
		logger.Error("初始化每日统计数据失败: %v", err)
	} else {
		logger.Info("每日统计数据初始化成功")
	}

	// 启动API密钥管理器
	key.StartKeyManager()
	logger.Info("API密钥管理器已启动")

	// 输出模型策略配置
	logModelStrategies()

	// 创建Gin路由
	router := gin.Default()

	// 设置API代理
	web.SetupApiProxy(router)

	// 设置API密钥管理
	web.SetupKeysAPI(router)

	// 设置Web界面
	web.SetupWebServer(router)

	// 保存端口到全局变量
	serverPort = cfg.Server.Port

	// 创建一个通道来接收信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在goroutine中启动服务器
	go func() {
		logger.Info("服务器启动在 :%d", serverPort)
		if err := router.Run(fmt.Sprintf(":%d", serverPort)); err != nil {
			logger.Error("服务器启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 等待服务器启动
	time.Sleep(500 * time.Millisecond)

	// 自动打开浏览器
	openBrowser(fmt.Sprintf("http://localhost:%d", serverPort))

	// 启动系统托盘
	go systray.Run(onReady, onExit)

	// 等待信号或退出通道
	select {
	case <-sigChan:
		logger.Info("接收到关闭信号，正在关闭服务器...")
	case <-quitChan:
		logger.Info("接收到退出请求，正在关闭服务器...")
	}

	// 保存API密钥
	config.SaveApiKeys()

	logger.Info("服务器已关闭")
}

// isConsolePresent 检测当前程序是否有控制台窗口（Linux版本）
func isConsolePresent() bool {
	// Linux下始终返回true，使用环境变量控制GUI模式
	return true
}

// logModelStrategies 输出所有已配置的模型策略
func logModelStrategies() {
	cfg := config.GetConfig()

	if len(cfg.App.ModelKeyStrategies) == 0 {
		logger.Info("未配置任何模型特定策略")
		return
	}

	logger.Info("===== 模型特定策略配置 =====")
	for model, strategyID := range cfg.App.ModelKeyStrategies {
		var strategyName string
		switch strategyID {
		case 1:
			strategyName = "高成功率"
		case 2:
			strategyName = "高分数"
		case 3:
			strategyName = "低RPM"
		case 4:
			strategyName = "低TPM"
		case 5:
			strategyName = "高余额"
		default:
			strategyName = "未知"
		}
		logger.Info("模型: %s, 策略: %s (%d)", model, strategyName, strategyID)
	}
	logger.Info("==========================")
}

// reloadConfig 重新加载配置文件
func reloadConfig() {
	configPath := getAbsolutePath("config/config.yaml")
	logger.Info("正在重新加载配置文件: %s", configPath)

	// 尝试加载配置文件
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("重新加载配置文件失败: %v", err)
		return
	}

	// 更新全局配置
	config.UpdateConfig(cfg)

	// 更新服务器端口
	serverPort = cfg.Server.Port

	// 输出模型策略配置
	logModelStrategies()

	logger.Info("配置文件重新加载成功")
}

// isAutoStartEnabled 检查是否已启用开机自动启动（Linux版本）
func isAutoStartEnabled() bool {
	// 检查用户的自启动目录是否存在启动项
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("获取用户主目录失败: %v", err)
		return false
	}

	autoStartPath := filepath.Join(homeDir, ".config/autostart/flowsilicon.desktop")
	_, err = os.Stat(autoStartPath)
	return err == nil
}

// enableAutoStart 启用开机自动启动（Linux版本）
func enableAutoStart() {
	// 获取当前可执行文件的绝对路径
	_, err := os.Executable()
	if err != nil {
		logger.Error("获取可执行文件路径失败: %v", err)
		return
	}

	// 确保目录存在
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("获取用户主目录失败: %v", err)
		return
	}

	autoStartDir := filepath.Join(homeDir, ".config/autostart")
	if err := os.MkdirAll(autoStartDir, 0755); err != nil {
		logger.Error("创建自启动目录失败: %v", err)
		return
	}

	// 创建桌面入口文件
	desktopEntry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=流动硅基 FlowSilicon
GenericName=API代理服务
Exec=%s --gui
Icon=%s
Comment=流动硅基API代理服务
Categories=Network;Utility;
Terminal=false
StartupNotify=true
StartupWMClass=flowsilicon
X-GNOME-Autostart-enabled=true
`, getAbsolutePath("start.sh"), getAbsolutePath("icons/hicolor/128x128/apps/flowsilicon.png"))

	autoStartPath := filepath.Join(autoStartDir, "flowsilicon.desktop")
	if err := os.WriteFile(autoStartPath, []byte(desktopEntry), 0644); err != nil {
		logger.Error("创建自启动文件失败: %v", err)
		return
	}

	logger.Info("已成功设置开机自动启动")
}

// disableAutoStart 禁用开机自动启动（Linux版本）
func disableAutoStart() {
	// 删除自启动文件
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("获取用户主目录失败: %v", err)
		return
	}

	autoStartPath := filepath.Join(homeDir, ".config/autostart/flowsilicon.desktop")
	if err := os.Remove(autoStartPath); err != nil && !os.IsNotExist(err) {
		logger.Error("删除自启动文件失败: %v", err)
		return
	}

	logger.Info("已成功禁用开机自动启动")
}

// minimizeIcon 最小化系统托盘图标（伪隐藏）
func minimizeIcon() {
	// 设置极小的图标
	systray.SetIcon(make([]byte, 0)) // 空图标
	systray.SetTitle("")
	systray.SetTooltip("流动硅基 FlowSilicon v" + Version + " (隐藏中)")
	logger.Info("已最小化系统托盘图标")
}

// restoreIcon 恢复系统托盘图标
func restoreIcon() {
	// 重新设置图标和标题
	icon := getTrayIcon()
	if len(icon) > 0 {
		systray.SetIcon(icon)
	}
	systray.SetTitle("流动硅基")
	systray.SetTooltip("流动硅基 FlowSilicon v" + Version)
	logger.Info("已恢复系统托盘图标")
}
