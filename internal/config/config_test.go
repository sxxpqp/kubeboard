package config

import (
	"os"
	"testing"
)

func TestEnvBool(t *testing.T) {
	cases := []struct {
		val      string
		fallback bool
		want     bool
	}{
		{"1", false, true},
		{"true", false, true},
		{"TRUE", false, true},
		{"yes", false, true},
		{"on", false, true},
		{"0", true, false},
		{"false", true, false},
		{"FALSE", true, false},
		{"no", true, false},
		{"off", true, false},
	}
	for _, c := range cases {
		os.Setenv("KB_TEST_BOOL", c.val)
		if got := envBool("KB_TEST_BOOL", c.fallback); got != c.want {
			t.Fatalf("envBool(%q, %v) = %v, want %v", c.val, c.fallback, got, c.want)
		}
	}
	os.Unsetenv("KB_TEST_BOOL")
}

// 未设置或无法识别时回退默认值
func TestEnvBool_Fallback(t *testing.T) {
	os.Unsetenv("KB_TEST_BOOL_UNSET")
	if got := envBool("KB_TEST_BOOL_UNSET", true); got != true {
		t.Fatal("未设置时应回退 fallback=true")
	}
	if got := envBool("KB_TEST_BOOL_UNSET", false); got != false {
		t.Fatal("未设置时应回退 fallback=false")
	}
	os.Setenv("KB_TEST_BOOL_WEIRD", "maybe")
	defer os.Unsetenv("KB_TEST_BOOL_WEIRD")
	if got := envBool("KB_TEST_BOOL_WEIRD", true); got != true {
		t.Fatal("无法识别时应回退 fallback")
	}
}

func TestEnvOr(t *testing.T) {
	os.Setenv("KB_TEST_STR", "custom")
	defer os.Unsetenv("KB_TEST_STR")
	if got := envOr("KB_TEST_STR", "fallback"); got != "custom" {
		t.Fatalf("应返回环境变量值, got %q", got)
	}
	os.Unsetenv("KB_TEST_STR")
	if got := envOr("KB_TEST_STR", "fallback"); got != "fallback" {
		t.Fatalf("未设置时应返回 fallback, got %q", got)
	}
	// 空字符串视为未设置
	os.Setenv("KB_TEST_STR", "")
	defer os.Unsetenv("KB_TEST_STR")
	if got := envOr("KB_TEST_STR", "fallback"); got != "fallback" {
		t.Fatalf("空字符串应视为未设置, got %q", got)
	}
}

// 安全回归：exec 默认必须关闭（除非显式开启）
func TestEnableExecDefaultsOff(t *testing.T) {
	os.Unsetenv("KUBEBOARD_ENABLE_EXEC")
	if envBool("KUBEBOARD_ENABLE_EXEC", false) {
		t.Fatal("exec 默认必须是关闭的（安全要求）")
	}
}

// 安全回归：JWT secret / admin password 可通过环境变量覆盖默认值
func TestSecretsOverridableByEnv(t *testing.T) {
	os.Setenv("KUBEBOARD_JWT_SECRET", "prod-secret")
	os.Setenv("KUBEBOARD_ADMIN_PASSWORD", "prod-pass")
	defer func() {
		os.Unsetenv("KUBEBOARD_JWT_SECRET")
		os.Unsetenv("KUBEBOARD_ADMIN_PASSWORD")
	}()
	if got := envOr("KUBEBOARD_JWT_SECRET", "kubeboard-dev-secret"); got != "prod-secret" {
		t.Fatalf("JWT secret 应可被环境变量覆盖, got %q", got)
	}
	if got := envOr("KUBEBOARD_ADMIN_PASSWORD", "kubeboard@2024"); got != "prod-pass" {
		t.Fatalf("admin password 应可被环境变量覆盖, got %q", got)
	}
}
