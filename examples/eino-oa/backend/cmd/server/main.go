package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wuhaitao/eino-oa/backend/config"
	"github.com/wuhaitao/eino-oa/backend/internal/agent"
	"github.com/wuhaitao/eino-oa/backend/internal/auth"
	"github.com/wuhaitao/eino-oa/backend/internal/handler"
	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
	"github.com/wuhaitao/eino-oa/backend/internal/service"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := model.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	studentRepo := repository.NewStudentRepo(db)
	courseRepo := repository.NewCourseRepo(db)
	gradeRepo := repository.NewGradeRepo(db)

	studentSvc := service.NewStudentService(studentRepo)
	courseSvc := service.NewCourseService(courseRepo)
	gradeSvc := service.NewGradeService(gradeRepo)
	sessionSvc := service.NewSessionService(repository.NewSessionRepo(db))

	// ---------- 鉴权 ----------
	authSvc := auth.NewService(repository.NewAuthRepo(db), cfg.SessionTTL)
	admin, err := authSvc.EnsureAdmin(cfg.AdminUsername, cfg.AdminPassword)
	if err != nil {
		log.Fatalf("初始化管理员失败: %v\n请在 backend/.env 中设置 ADMIN_USERNAME 和 ADMIN_PASSWORD", err)
	}
	log.Printf("管理员账号 %q 已就绪（密码由环境变量 ADMIN_PASSWORD 管理，修改后重启生效）", admin.Username)
	authSvc.Cleanup()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			authSvc.Cleanup()
		}
	}()

	studentHandler := handler.NewStudentHandler(studentSvc)
	courseHandler := handler.NewCourseHandler(courseSvc)
	gradeHandler := handler.NewGradeHandler(gradeSvc)
	authHandler := handler.NewAuthHandler(authSvc)
	oauthHandler := handler.NewOAuthHandler(authSvc)

	r := gin.Default()
	// 不信任任意 X-Forwarded-For，登录限流按真实连接地址计数。
	_ = r.SetTrustedProxies(nil)

	// 基础安全响应头。CSP 由 OAuth 授权页单独设置（页面为纯内联样式）。
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	})

	// 技能文档对外公开（不含任何密钥），方便 Agent 直接下载。
	if dir := findDir("skills", "../skills", "../../skills"); dir != "" {
		r.Use(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/skills/") &&
				strings.EqualFold(filepath.Ext(c.Request.URL.Path), ".md") {
				c.Header("Content-Type", "text/markdown; charset=utf-8")
			}
			c.Next()
		})
		r.StaticFS("/skills", http.Dir(dir))
		log.Printf("技能文档目录: %s", dir)
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ---------- API ----------
	api := r.Group("/api")
	authHandler.RegisterPublic(api)

	protected := api.Group("")
	protected.Use(authSvc.RequireAuth())
	authHandler.RegisterProtected(protected)
	studentHandler.Register(protected)
	courseHandler.Register(protected)
	gradeHandler.Register(protected)

	// ---------- AI Agent ----------
	var chatAgent *agent.ChatAgent
	if cfg.DeepSeekKey == "" {
		log.Println("未设置 DEEPSEEK_API_KEY，AI Agent 未启用（对话接口返回 503）")
	} else {
		chatAgent, err = agent.NewChatAgent(ctx, cfg.DeepSeekKey, cfg.DeepSeekModel, cfg.DeepSeekBase, studentSvc, courseSvc, gradeSvc)
		if err != nil {
			log.Printf("AI Agent 初始化失败: %v", err)
			chatAgent = nil
		} else {
			log.Println("AI Agent 已启用")
		}
	}
	// 无论 AI 是否可用都注册会话路由，前端可以正常创建会话并看到明确的 503 提示。
	chatHandler := handler.NewChatHandler(chatAgent, sessionSvc)
	chatHandler.Register(protected)

	// ---------- OAuth2 授权服务器 ----------
	oauthHandler.Register(r)

	// ---------- 可选：由 Go 直接托管前端构建产物 ----------
	if dist := findDir("frontend/dist", "../frontend/dist"); dist != "" {
		serveFrontend(r, dist)
		log.Printf("前端静态目录: %s", dist)
	}

	log.Printf("服务启动在 :%s（首次访问请先登录）", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

func findDir(candidates ...string) string {
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

// serveFrontend 托管 Vite 构建产物，并对前端路由做 index.html 回退。
func serveFrontend(r *gin.Engine, dist string) {
	fs := http.Dir(dist)
	fileServer := http.FileServer(fs)
	r.Static("/assets", filepath.Join(dist, "assets"))
	for _, name := range []string{"favicon.svg", "icons.svg", "robots.txt"} {
		if _, err := os.Stat(filepath.Join(dist, name)); err == nil {
			r.StaticFile("/"+name, filepath.Join(dist, name))
		}
	}
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		for _, prefix := range []string{"/api", "/oauth", "/.well-known", "/skills"} {
			if p == prefix || strings.HasPrefix(p, prefix+"/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
		}
		if f, err := fs.Open(p); err == nil {
			defer f.Close()
			if st, err := f.Stat(); err == nil && !st.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		c.Header("Cache-Control", "no-cache")
		http.ServeFile(c.Writer, c.Request, filepath.Join(dist, "index.html"))
	})
}
