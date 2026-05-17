package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(handler *Handler, webFS embed.FS) *gin.Engine {
	// 创建 Gin 引擎
	r := gin.Default()

	// API 路由组
	api := r.Group("/api")
	{
		api.POST("/upload", handler.UploadIDL)         // 上传 IDL 文件
		api.POST("/test", handler.RunTest)             // 执行测试（一次性返回）
		api.POST("/test/stream", handler.RunTestStream) // 执行测试（SSE 流式返回）
	}

	// 静态文件服务(使用嵌入的文件系统)
	webSubFS, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}

	// 使用 StaticFS 提供静态文件服务
	r.StaticFS("/static", http.FS(webSubFS))

	// 首页重定向到 index.html
	r.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(webSubFS, "index.html")
		if err != nil {
			c.String(500, "Error loading page: %v", err)
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	})

	// 直接访问 index.html
	r.GET("/index.html", func(c *gin.Context) {
		data, err := fs.ReadFile(webSubFS, "index.html")
		if err != nil {
			c.String(500, "Error loading page: %v", err)
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	})

	// Vue.js 文件
	r.GET("/vue.global.js", func(c *gin.Context) {
		data, err := fs.ReadFile(webSubFS, "vue.global.js")
		if err != nil {
			c.String(500, "Error loading Vue.js: %v", err)
			return
		}
		c.Data(200, "application/javascript; charset=utf-8", data)
	})

	// Pico.css 文件
	r.GET("/pico.min.css", func(c *gin.Context) {
		data, err := fs.ReadFile(webSubFS, "pico.min.css")
		if err != nil {
			c.String(500, "Error loading Pico.css: %v", err)
			return
		}
		c.Data(200, "text/css; charset=utf-8", data)
	})

	return r
}
