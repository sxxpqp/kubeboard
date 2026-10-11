package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/sxxpqp/kubeboard/internal/api"
	"github.com/sxxpqp/kubeboard/internal/auth"
	"github.com/sxxpqp/kubeboard/internal/config"
	"github.com/sxxpqp/kubeboard/internal/k8s"
)

func main() {
	cfg := config.Load()

	k8sClient, err := k8s.New(cfg.KubeConfig)
	if err != nil {
		log.Fatalf("connect to kubernetes: %v", err)
	}
	log.Printf("connected to cluster, version: %s, metrics-server: %v", k8sClient.Version, k8sClient.Metrics != nil)

	authMgr := auth.NewManager(cfg.JWTSecret, cfg.AdminUser, cfg.AdminPassword)

	if env := os.Getenv("GIN_MODE"); env == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	api.New(authMgr, k8sClient, cfg.EnableExec).Register(r)

	if cfg.StaticDir != "" {
		serveStatic(r, cfg.StaticDir)
	}

	log.Printf("kubeboard listening on %s", cfg.ListenAddr)
	if err := r.Run(cfg.ListenAddr); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

// serveStatic 托管前端构建产物，非 /api 路径回退到 index.html（SPA history 路由）。
func serveStatic(r *gin.Engine, dir string) {
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		full := dir + path
		if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
			c.File(full)
			return
		}
		c.File(dir + "/index.html")
	})
}
