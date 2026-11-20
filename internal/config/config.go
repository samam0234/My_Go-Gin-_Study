package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 는 프로세스 환경 변수에서 읽은 실행 설정이다.
// 비밀값은 코드에 적지 않고, 비어 있으면 개발용 기본값만 쓴다.
type Config struct {
	Env                string
	Addr               string
	Driver             string
	DSN                string
	JWTSecret          string
	JWTTTL             time.Duration
	CORSOrigin         string
	AIProvider         string
	AIModel            string
	AIKey              string
	OllamaURL          string
	AIDailyLimit       int
	MCPToken           string
	UploadDir          string
	MaxUpload          int64
	WebDir             string
	OpenAPI            string
	Seed               bool
	RatePerMin         int
	OAuthMock          bool
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirect     string
}

func Load() Config {
	_ = godotenv.Load(".env")
	env := getenv("APP_ENV", "development")
	ttl, err := time.ParseDuration(getenv("JWT_TTL", "168h"))
	if err != nil || ttl <= 0 {
		ttl = 168 * time.Hour
	}
	oauth := os.Getenv("OAUTH_MOCK")
	oauthMock := env == "development"
	if oauth == "1" {
		oauthMock = true
	}
	if oauth == "0" {
		oauthMock = false
	}
	return Config{
		Env:                env,
		Addr:               getenv("HTTP_ADDR", ":8080"),
		Driver:             getenv("DB_DRIVER", "sqlite"),
		DSN:                os.Getenv("DB_DSN"),
		JWTSecret:          getenv("JWT_SECRET", "dev-only-change-me-please-32b"),
		JWTTTL:             ttl,
		CORSOrigin:         getenv("CORS_ORIGIN", "http://localhost:5173"),
		AIProvider:         getenv("AI_PROVIDER", "mock"),
		AIModel:            getenv("AI_MODEL", "mock"),
		AIKey:              os.Getenv("ANTHROPIC_API_KEY"),
		OllamaURL:          getenv("OLLAMA_URL", "http://localhost:11434"),
		AIDailyLimit:       atoi(getenv("AI_DAILY_TOKEN_LIMIT", "200000")),
		MCPToken:           getenv("MCP_TOKEN", "dev-mcp-token"),
		UploadDir:          getenv("UPLOAD_DIR", "uploads"),
		MaxUpload:          atoi64(getenv("MAX_UPLOAD_BYTES", "5242880")),
		WebDir:             os.Getenv("WEB_DIR"),
		OpenAPI:            os.Getenv("OPENAPI_PATH"),
		Seed:               os.Getenv("SEED") == "1",
		RatePerMin:         atoi(getenv("RATE_LIMIT_PER_MIN", "0")),
		OAuthMock:          oauthMock,
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirect:     getenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/oauth/google/callback"),
	}
}

func (c Config) Validate() error {
	if len(c.JWTSecret) < 16 {
		return errors.New("JWT_SECRET는 16자 이상이어야 합니다")
	}
	if c.Env == "production" && strings.Contains(c.JWTSecret, "change-me") {
		return errors.New("운영 환경에서는 JWT_SECRET를 예시값이 아닌 값으로 바꾸세요")
	}
	if c.MaxUpload < 1 {
		return errors.New("MAX_UPLOAD_BYTES는 1 이상이어야 합니다")
	}
	return nil
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func atoi64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
