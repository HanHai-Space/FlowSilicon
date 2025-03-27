/*
*
@author: Hanhai
@since: 2025/3/16 21:57:52
@desc:
*
*/
package web

import (
	"embed"
	"flowsilicon/internal/config"
	"flowsilicon/internal/proxy"
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/css/* static/js/* static/img/*
var staticFS embed.FS

// SetupApiProxy 设置 API 代理路由
func SetupApiProxy(router *gin.Engine) {
	// 代理所有 API 请求
	router.Any("/api/*path", proxy.HandleApiProxy)

	// 添加对 OpenAI 格式 API 的支持
	router.Any("/v1/*path", proxy.HandleOpenAIProxy)

	// 添加对无版本号路径的支持
	// 聊天完成
	router.Any("/chat", proxy.HandleOpenAIProxy)
	router.Any("/chat/*path", proxy.HandleOpenAIProxy)

	// 文本完成
	router.Any("/completions", proxy.HandleOpenAIProxy)

	// 嵌入
	router.Any("/embeddings", proxy.HandleOpenAIProxy)

	// 图像生成
	router.Any("/images", proxy.HandleOpenAIProxy)
	router.Any("/images/*path", proxy.HandleOpenAIProxy)

	// 模型列表
	router.Any("/models", proxy.HandleOpenAIProxy)

	// 重排序
	router.Any("/rerank", proxy.HandleOpenAIProxy)

	// 用户信息
	router.Any("/user/info", proxy.HandleOpenAIProxy)
}

// SetupKeysAPI 设置API密钥相关路由
func SetupKeysAPI(router *gin.Engine) {
	// 获取当前请求统计
	router.GET("/request-stats/current", handleGetCurrentRequestStats)

	// 获取每日统计数据
	router.GET("/request-stats/daily", handleGetDailyStats)

	// 获取指定日期的统计数据
	router.GET("/request-stats/daily/:date", handleGetDailyStatsByDate)

	// 刷新所有API密钥余额
	router.POST("/keys/refresh", handleRefreshAllKeysBalance)
}

// SetupWebServer 设置 Web 服务器
func SetupWebServer(router *gin.Engine) {
	// 添加禁用静态文件缓存的中间件
	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static-fs/") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})

	// 加载模板
	templ := template.Must(template.New("").ParseFS(templatesFS, "templates/*.html"))
	router.SetHTMLTemplate(templ)

	// 静态文件 - 使用嵌入式文件系统
	router.StaticFS("/static", http.FS(staticFS))

	// 静态文件 - 直接从文件系统提供
	router.Static("/static-fs", "./web/static")

	// 网站图标
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static-fs/img/favicon_32.ico")
	})

	// 页面路由
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":                  config.GetConfig().App.Title,
			"max_balance_display":    config.GetConfig().App.MaxBalanceDisplay,
			"items_per_page":         config.GetConfig().App.ItemsPerPage,
			"auto_update_interval":   config.GetConfig().App.AutoUpdateInterval,
			"stats_refresh_interval": config.GetConfig().App.StatsRefreshInterval,
			"rate_refresh_interval":  config.GetConfig().App.RateRefreshInterval,
		})
	})

	// 设置页面
	router.GET("/setting", func(c *gin.Context) {
		c.HTML(http.StatusOK, "setting.html", gin.H{
			"title": config.GetConfig().App.Title,
		})
	})

	// 模型管理页面
	router.GET("/model", handleModelManagementPage)

	// API 密钥管理
	router.GET("/keys", handleListKeys)
	router.POST("/keys", handleAddKey)
	router.DELETE("/keys/:key", handleDeleteKey)
	router.POST("/keys/batch", handleBatchAddKeys)
	router.POST("/keys/check", handleCheckKey)
	router.POST("/keys/mode", handleSetKeyMode)
	router.GET("/keys/mode", handleGetKeyMode)
	router.POST("/keys/:key/enable", handleEnableKey)
	router.POST("/keys/:key/disable", handleDisableKey)
	router.DELETE("/keys/zero-balance", handleDeleteZeroBalanceKeys)
	router.DELETE("/keys/low-balance/:threshold", handleDeleteLowBalanceKeys)
	router.GET("/test-key", handleGetTestKey)

	// 设置页面的-模型管理API
	router.GET("/models/list", getModelsHandler)
	router.POST("/models/sync", syncModelsHandler)
	router.POST("/models/strategy", updateModelStrategyHandler)
	router.DELETE("/models/strategy", deleteModelStrategyHandler)

	// 模型管理页面-模型管理API
	router.GET("/models-api/list", getModelsAPIHandler)
	router.GET("/models-api/status", getModelsStatusHandler)
	router.POST("/models-api/update", updateModelsHandler)
	router.POST("/models-api/type", updateModelTypeHandler)

	// API 密钥统计
	router.GET("/stats", handleStats)

	// 日志查看
	router.GET("/logs", handleGetLogs)

	// 测试embeddings API
	router.POST("/test-chat", handleTestChat)

	// 测试embeddings API
	router.POST("/test-embeddings", handleTestEmbeddings)

	// 测试图片生成API
	router.POST("/test-images", handleTestImages)

	// 测试模型列表API
	router.POST("/test-models", handleTestModels)

	// 测试重排序API
	router.POST("/test-rerank", handleTestRerank)

	// 请求统计数据
	router.GET("/request-stats", handleRequestStats)

	// 设置相关API
	router.GET("/settings/config", handleGetSettings)
	router.POST("/settings/config", handleSaveSettings)

	// 系统重启API
	router.POST("/system/restart", handleSystemRestart)

	// API密钥代理 - 解决CORS问题
	router.GET("/proxy/apikeys", handleApiKeyProxy)
}
