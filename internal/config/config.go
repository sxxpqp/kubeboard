package config

import (
	"flag"
	"os"
)

type Config struct {
	ListenAddr    string
	KubeConfig    string
	JWTSecret     string
	StaticDir     string
	AdminUser     string
	AdminPassword string
	EnableExec    bool
}

func Load() *Config {
	c := &Config{}
	flag.StringVar(&c.ListenAddr, "listen", ":9090", "HTTP listen address")
	flag.StringVar(&c.KubeConfig, "kubeconfig", "", "path to kubeconfig, empty for in-cluster")
	flag.StringVar(&c.JWTSecret, "jwt-secret", envOr("KUBEBOARD_JWT_SECRET", "kubeboard-dev-secret"), "JWT signing secret")
	flag.StringVar(&c.StaticDir, "static", "", "directory of built frontend assets, empty to disable")
	flag.StringVar(&c.AdminUser, "admin-user", "admin", "initial admin username")
	flag.StringVar(&c.AdminPassword, "admin-password", envOr("KUBEBOARD_ADMIN_PASSWORD", "kubeboard@2024"), "initial admin password")
	flag.BoolVar(&c.EnableExec, "enable-exec", envBool("KUBEBOARD_ENABLE_EXEC", false), "enable interactive pod terminal (requires pods/exec RBAC)")
	flag.Parse()
	return c
}

func envBool(key string, fallback bool) bool {
	switch os.Getenv(key) {
	case "1", "true", "TRUE", "yes", "on":
		return true
	case "0", "false", "FALSE", "no", "off":
		return false
	}
	return fallback
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
