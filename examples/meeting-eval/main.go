package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"meetingeval/internal/eval"
)

//go:embed web
var webFiles embed.FS

func main() {
	port := flag.String("port", "", "监听端口（默认读取 EVAL_PORT，否则 8765）")
	flag.Parse()

	cfg := eval.LoadConfig()
	if *port != "" {
		cfg.Port = *port
	}

	store, err := eval.OpenStore(filepath.Join(cfg.DataDir, "store.json"))
	if err != nil {
		log.Fatalf("初始化存储失败：%v", err)
	}

	server := eval.NewServer(cfg, store)
	mux := http.NewServeMux()
	server.Register(mux)

	webRoot, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatalf("加载前端资源失败：%v", err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(webRoot)))

	httpServer := &http.Server{
		Addr:              cfg.Host + ":" + cfg.Port,
		Handler:           logging(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("会议助手评测台已启动：http://%s:%s（模式：%s，模型：%s）", cfg.Host, cfg.Port, cfg.Mode, cfg.Model)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务启动失败：%v", err)
	}
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		if r.URL.Path != "/api/health" {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
		}
	})
}
