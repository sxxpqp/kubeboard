package auth

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// 登录限流阈值。按 username 与 client IP 双维度统计，
// 避免单一用户名被锁导致整站唯一账号不可用（DoS）。
const (
	maxFailuresPerUser = 5
	maxFailuresPerIP   = 20
	lockDuration       = 5 * time.Minute
)

type counter struct {
	fails int
	until time.Time
}

type Manager struct {
	secret   []byte
	user     string
	passHash []byte

	mu     sync.Mutex
	byUser map[string]*counter
	byIP   map[string]*counter
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var ErrInvalidCredential = errors.New("invalid username or password")
var ErrLocked = errors.New("too many failed attempts, please try again later")

func NewManager(secret, user, password string) *Manager {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return &Manager{
		secret:   []byte(secret),
		user:     user,
		passHash: hash,
		byUser:   map[string]*counter{},
		byIP:     map[string]*counter{},
	}
}

// locked 判定给定 key 是否处于锁定窗口内（只读，不修改状态）。
// 注意：不能用它做惰性清理，否则会在每次调用顶部误删未锁定的计数器。
func locked(m map[string]*counter, key string) bool {
	c, ok := m[key]
	if !ok {
		return false
	}
	return time.Now().Before(c.until)
}

// recordFailure 累计失败次数，达到阈值则锁定窗口（调用方需持锁）。
// 锁定窗口到期后由 locked() 判定失效，此处仅在未锁定时累加。
func recordFailure(m map[string]*counter, key string, limit int) {
	c, ok := m[key]
	if !ok {
		c = &counter{}
		m[key] = c
	}
	c.fails++
	if c.fails >= limit {
		c.fails = 0
		c.until = time.Now().Add(lockDuration)
	}
}

// Login 校验凭据并签发令牌。ip 用于按来源地址限流。
// 两个维度（username / ip）独立累计：任一个触发锁定都会拒绝本次登录，
// 但另一个维度仍会记录本次失败，避免攻击者靠切换用户名绕过 IP 限流。
func (m *Manager) Login(username, password, ip string) (string, error) {
	m.mu.Lock()
	userLocked := locked(m.byUser, username)
	ipLocked := locked(m.byIP, ip)
	m.mu.Unlock()

	if userLocked || ipLocked {
		// 仍记录另一维度的失败，防绕过。
		m.mu.Lock()
		if !userLocked {
			recordFailure(m.byUser, username, maxFailuresPerUser)
		}
		if !ipLocked {
			recordFailure(m.byIP, ip, maxFailuresPerIP)
		}
		m.mu.Unlock()
		log.Printf("[auth] login blocked (rate limited): user=%q ip=%s", username, ip)
		return "", ErrLocked
	}

	if username != m.user || bcrypt.CompareHashAndPassword(m.passHash, []byte(password)) != nil {
		m.mu.Lock()
		recordFailure(m.byUser, username, maxFailuresPerUser)
		recordFailure(m.byIP, ip, maxFailuresPerIP)
		m.mu.Unlock()
		log.Printf("[auth] login failed: user=%q ip=%s", username, ip)
		return "", ErrInvalidCredential
	}

	m.mu.Lock()
	delete(m.byUser, username)
	delete(m.byIP, ip)
	m.mu.Unlock()

	log.Printf("[auth] login success: user=%q ip=%s", username, ip)
	return m.sign(username)
}

func (m *Manager) sign(username string) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "kubeboard",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
