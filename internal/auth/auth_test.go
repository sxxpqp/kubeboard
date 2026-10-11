package auth

import (
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testUser = "admin"
	testPass = "s3cret-pass"
	testIP   = "10.0.0.1"
)

func newTestManager() *Manager {
	return NewManager("test-secret-key", testUser, testPass)
}

// ==================== 凭据校验 ====================

func TestLogin_Success(t *testing.T) {
	m := newTestManager()
	tok, err := m.Login(testUser, testPass, testIP)
	if err != nil {
		t.Fatalf("登录应成功，却返回错误: %v", err)
	}
	if tok == "" {
		t.Fatal("成功登录应返回非空 token")
	}
	claims, err := m.Verify(tok)
	if err != nil {
		t.Fatalf("刚签发的 token 应可验证: %v", err)
	}
	if claims.Username != testUser {
		t.Fatalf("token 中用户名不符: got %q want %q", claims.Username, testUser)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	m := newTestManager()
	if _, err := m.Login(testUser, "wrong-pass", testIP); err != ErrInvalidCredential {
		t.Fatalf("错误密码应返回 ErrInvalidCredential, got %v", err)
	}
}

func TestLogin_WrongUsername(t *testing.T) {
	m := newTestManager()
	if _, err := m.Login("nonexistent", testPass, testIP); err != ErrInvalidCredential {
		t.Fatalf("错误用户名应返回 ErrInvalidCredential, got %v", err)
	}
}

// ==================== 按 username 维度限流 ====================

func TestRateLimit_ByUser_LocksAfterThreshold(t *testing.T) {
	m := newTestManager()
	// 前 maxFailuresPerUser-1 次：仍是凭据错误
	for i := 0; i < maxFailuresPerUser-1; i++ {
		if _, err := m.Login(testUser, "bad", testIP); err != ErrInvalidCredential {
			t.Fatalf("第 %d 次失败应返回 ErrInvalidCredential, got %v", i+1, err)
		}
	}
	// 第 maxFailuresPerUser 次：触发锁定
	if _, err := m.Login(testUser, "bad", testIP); err != ErrInvalidCredential {
		t.Fatalf("达到阈值的那次仍应返回凭据错误, got %v", err)
	}
	// 之后即便密码正确也被拒
	if _, err := m.Login(testUser, testPass, testIP); err != ErrLocked {
		t.Fatalf("锁定后正确密码也应返回 ErrLocked, got %v", err)
	}
}

func TestRateLimit_DoesNotLockBeforeThreshold(t *testing.T) {
	m := newTestManager()
	for i := 0; i < maxFailuresPerUser-1; i++ {
		_, _ = m.Login(testUser, "bad", testIP)
	}
	// 未达阈值，正确密码应可登录（验证 locked() 未误清/误判）
	if _, err := m.Login(testUser, testPass, testIP); err != nil {
		t.Fatalf("未达阈值时正确密码应可登录, got %v", err)
	}
}

func TestLoginSuccess_ResetsCounters(t *testing.T) {
	m := newTestManager()
	for i := 0; i < maxFailuresPerUser-1; i++ {
		_, _ = m.Login(testUser, "bad", testIP)
	}
	if _, err := m.Login(testUser, testPass, testIP); err != nil {
		t.Fatalf("成功登录应清空计数器: %v", err)
	}
	// 再失败 maxFailuresPerUser-1 次也不应被锁（计数已重置）
	for i := 0; i < maxFailuresPerUser-1; i++ {
		if _, err := m.Login(testUser, "bad", testIP); err != ErrInvalidCredential {
			t.Fatalf("重置后第 %d 次失败不应被锁, got %v", i+1, err)
		}
	}
}

// ==================== 按 IP 维度限流（独立累计）====================

// 关键回归：IP 维度不能被 user 维度短路。
// 攻击者用不同用户名、同一 IP 撞库时，user 维度永不触发，必须靠 IP 维度拦截。
func TestRateLimit_ByIP_IndependentOfUser(t *testing.T) {
	m := newTestManager()
	// 用不同用户名（每个只失败 1 次），累计到 IP 阈值
	for i := 0; i < maxFailuresPerIP; i++ {
		u := "attacker-" + strings.Repeat("x", i%7) + string(rune('a'+i%26)) + itoa(i)
		if _, err := m.Login(u, "bad", testIP); err != ErrInvalidCredential {
			t.Fatalf("第 %d 次不同用户名失败应返回凭据错误, got %v", i+1, err)
		}
	}
	// IP 已锁定：换新用户名 + 正确密码也应被拒
	if _, err := m.Login(testUser, testPass, testIP); err != ErrLocked {
		t.Fatalf("IP 达阈值后应锁定, got %v", err)
	}
	// 换 IP 不受影响
	if _, err := m.Login(testUser, testPass, "10.0.0.99"); err != nil {
		t.Fatalf("其它 IP 不应受限, got %v", err)
	}
}

// 关键回归：一个维度已锁定时，另一维度仍须记录失败（防绕过）。
func TestRateLimit_OtherDimensionStillRecords(t *testing.T) {
	m := newTestManager()
	// 先锁死 username 维度
	for i := 0; i < maxFailuresPerUser; i++ {
		_, _ = m.Login(testUser, "bad", testIP)
	}
	if _, err := m.Login(testUser, testPass, testIP); err != ErrLocked {
		t.Fatalf("user 维度应已锁定: %v", err)
	}
	// user 已锁，此后每次尝试应继续累加 IP 计数
	// 用同 IP 换用户名触发记录：需要 IP 计数在增长
	before := m.byIP[testIP].fails
	_, _ = m.Login(testUser, "bad", testIP) // user 锁定 -> 走 blocked 分支，记录 IP
	if m.byIP[testIP].fails <= before {
		t.Fatalf("user 锁定时 IP 维度未继续累计: before=%d after=%d", before, m.byIP[testIP].fails)
	}
}

// ==================== 锁定窗口 ====================

func TestLocked_ExpiresAfterWindow(t *testing.T) {
	m := newTestManager()
	m.mu.Lock()
	m.byUser[testUser] = &counter{fails: 0, until: time.Now().Add(-time.Second)} // 已过期
	m.mu.Unlock()
	if locked(m.byUser, testUser) {
		t.Fatal("锁定窗口已过期，locked() 应返回 false")
	}
}

func TestLocked_ZeroTimeIsNotLocked(t *testing.T) {
	// 回归：曾用 time.Now().After(zeroTime) 判定，导致零值计数被误判为锁定
	m := newTestManager()
	m.mu.Lock()
	m.byUser[testUser] = &counter{fails: 1} // until 为零值
	m.mu.Unlock()
	if locked(m.byUser, testUser) {
		t.Fatal("until 为零值的计数器不应被判为锁定")
	}
}

func TestLocked_UnknownKey(t *testing.T) {
	m := newTestManager()
	if locked(m.byUser, "never-seen") {
		t.Fatal("未知 key 不应被判为锁定")
	}
}

// recordFailure 达到阈值后应重置 fails 并设置 until
func TestRecordFailure_SetsLockWindow(t *testing.T) {
	m := make(map[string]*counter)
	for i := 0; i < maxFailuresPerUser; i++ {
		recordFailure(m, "k", maxFailuresPerUser)
	}
	c := m["k"]
	if c.fails != 0 {
		t.Fatalf("达阈值后 fails 应重置为 0, got %d", c.fails)
	}
	if !time.Now().Before(c.until) {
		t.Fatal("达阈值后 until 应在未来")
	}
}

// ==================== 并发安全 ====================

func TestLogin_ConcurrentSafe(t *testing.T) {
	m := newTestManager()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ip := "10.0.1." + itoa(i%10)
			if i%2 == 0 {
				_, _ = m.Login(testUser, "bad", ip)
			} else {
				_, _ = m.Login(testUser, testPass, ip)
			}
		}(i)
	}
	wg.Wait()
	// 只验证不 panic / 不死锁；再用 -race 检查数据竞争
	if _, err := m.Login("clean-user", testPass, "192.168.0.1"); err != ErrInvalidCredential {
		t.Fatalf("无关用户不应受影响: %v", err)
	}
}

// ==================== JWT 签发与验证 ====================

func TestVerify_RejectsTamperedToken(t *testing.T) {
	m := newTestManager()
	tok, _ := m.Login(testUser, testPass, testIP)
	// 篡改签名部分
	parts := strings.Split(tok, ".")
	parts[2] = "tampered-signature"
	if _, err := m.Verify(strings.Join(parts, ".")); err == nil {
		t.Fatal("篡改签名的 token 应验证失败")
	}
}

func TestVerify_RejectsWrongSecret(t *testing.T) {
	m1 := NewManager("secret-A", testUser, testPass)
	m2 := NewManager("secret-B", testUser, testPass)
	tok, _ := m1.Login(testUser, testPass, testIP)
	if _, err := m2.Verify(tok); err == nil {
		t.Fatal("用其它密钥签发的 token 应验证失败")
	}
}

func TestVerify_RejectsGarbage(t *testing.T) {
	m := newTestManager()
	for _, bad := range []string{"", "not-a-jwt", "a.b.c", ".."} {
		if _, err := m.Verify(bad); err == nil {
			t.Fatalf("非法 token %q 应验证失败", bad)
		}
	}
}

func TestVerify_RejectsAlgNone(t *testing.T) {
	// 经典 JWT 攻击：alg=none 的伪造 token 必须被拒
	m := newTestManager()
	// base64url({"alg":"none","typ":"JWT"}).base64url({"username":"admin"})
	noneTok := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VybmFtZSI6ImFkbWluIn0."
	if _, err := m.Verify(noneTok); err == nil {
		t.Fatal("alg=none 的 token 应验证失败")
	}
}

func TestSign_ExpiryIs24h(t *testing.T) {
	m := newTestManager()
	tok, _ := m.Login(testUser, testPass, testIP)
	claims, err := m.Verify(tok)
	if err != nil {
		t.Fatalf("token 应可验证: %v", err)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("token 应设置过期时间")
	}
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining < 23*time.Hour || remaining > 25*time.Hour {
		t.Fatalf("过期时间应约为 24 小时, got %v", remaining)
	}
	if claims.Issuer != "kubeboard" {
		t.Fatalf("Issuer 应为 kubeboard, got %q", claims.Issuer)
	}
}

// 工具：避免引入 strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := []byte{}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}
