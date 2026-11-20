package httpapi

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/samam0234/my-go-gin-study/internal/ai"
	"github.com/samam0234/my-go-gin-study/internal/auth"
	"github.com/samam0234/my-go-gin-study/internal/cache"
	"github.com/samam0234/my-go-gin-study/internal/config"
	"github.com/samam0234/my-go-gin-study/internal/notify"
	"github.com/samam0234/my-go-gin-study/internal/store"
)

const userKey = "authUser"

type Mailer interface {
	Send(to, subject, body string) error
}

type LogMailer struct{}

func (LogMailer) Send(to, subject, body string) error {
	log.Printf("mail to=%s subject=%s", to, subject)
	return nil
}

type googleProfile struct {
	Email    string
	Nickname string
}

type oauthPending struct {
	verifier string
	exp      time.Time
}

type Server struct {
	cfg            config.Config
	store          *store.Store
	cache          *cache.TTL
	hub            *notify.Hub
	idem           *idemStore
	mailer         Mailer
	ai             *ai.Client
	engine         *gin.Engine
	webDir         string
	openAPI        string
	upgrader       websocket.Upgrader
	oauthMu        sync.Mutex
	oauth          map[string]oauthPending
	exchangeGoogle func(ctx context.Context, code, verifier string) (googleProfile, error)
}

func New(cfg config.Config, st *store.Store) *Server {
	switch cfg.Env {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	}
	s := &Server{
		cfg:     cfg,
		store:   st,
		cache:   cache.New(),
		hub:     notify.New(),
		idem:    newIdem(),
		mailer:  LogMailer{},
		ai:      ai.New(cfg.AIProvider, cfg.AIModel, cfg.AIKey, cfg.OllamaURL),
		webDir:  locateDir(cfg.WebDir, "project/web", "web"),
		openAPI: locateFile(cfg.OpenAPI, "project/openapi.yaml", "openapi.yaml"),
		oauth:   map[string]oauthPending{},
	}
	s.exchangeGoogle = s.exchangeGoogleHTTP
	s.upgrader = websocket.Upgrader{CheckOrigin: s.allowOrigin}
	s.engine = s.routes()
	return s
}

func (s *Server) Router() http.Handler { return s.engine }

func (s *Server) routes() *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies([]string{})
	r.Use(requestID(), securityHeaders(), gin.Recovery(), s.cors(), newLimiter().middleware(s.cfg.RatePerMin))
	if s.cfg.Env != "test" {
		r.Use(accessLog())
	}
	r.GET("/health", s.health)
	r.GET("/", s.home)
	r.GET("/api-docs", s.docs)
	r.GET("/api-docs/openapi.yaml", s.openAPIFile)
	if s.webDir != "" {
		r.StaticFile("/app.js", filepath.Join(s.webDir, "app.js"))
		r.StaticFile("/app.css", filepath.Join(s.webDir, "app.css"))
	}
	r.GET("/ws", s.socket)
	r.POST("/mcp", s.mcpHTTP)
	r.GET("/uploads/:name", s.download)

	r.POST("/api/auth/signup", s.signup)
	r.POST("/api/auth/login", s.login)
	r.GET("/api/auth/oauth/google/start", s.oauthStart)
	r.GET("/api/auth/oauth/google/callback", s.oauthCallback)
	r.GET("/api/posts", s.listPosts)
	r.GET("/api/posts/:id", s.getPost)
	r.GET("/api/posts/:id/comments", s.listComments)

	authn := r.Group("/api", s.requireAuth)
	authn.POST("/auth/logout", s.logout)
	authn.GET("/auth/me", s.me)
	authn.POST("/posts", s.createPost)
	authn.PATCH("/posts/:id", s.patchPost)
	authn.DELETE("/posts/:id", s.deletePost)
	authn.POST("/posts/:id/comments", s.createComment)
	authn.DELETE("/comments/:id", s.deleteComment)
	authn.POST("/documents", s.createDocument)
	authn.GET("/documents", s.listDocuments)
	authn.GET("/documents/:id", s.getDocument)
	authn.POST("/documents/:id/questions", s.askDocument)
	authn.GET("/documents/:id/questions", s.listQuestions)
	authn.POST("/ai/chat", s.chat)
	authn.POST("/ai/chat/stream", s.chatStream)
	authn.POST("/uploads", s.upload)
	authn.POST("/payments", s.createPayment)
	authn.POST("/payments/:id/confirm", s.confirmPayment)
	authn.POST("/mail/test", s.mailTest)

	r.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "경로를 찾을 수 없습니다.", nil)
	})
	return r
}

func (s *Server) cors() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     splitOrigins(s.cfg.CORSOrigin),
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-Id", "Idempotency-Key"},
		ExposeHeaders:    []string{"X-Request-Id", "Location", "X-Cache", "Idempotency-Replayed"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}

func splitOrigins(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" || part == "*" {
			continue
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return []string{"http://localhost:5173"}
	}
	return out
}

func (s *Server) health(c *gin.Context) {
	if err := s.store.Ping(c.Request.Context()); err != nil {
		s.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) home(c *gin.Context) {
	if s.webDir == "" {
		c.JSON(http.StatusOK, gin.H{"service": "gin-study", "docs": "/api-docs", "health": "/health"})
		return
	}
	c.File(filepath.Join(s.webDir, "index.html"))
}

func (s *Server) docs(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, docsHTML)
}

func (s *Server) openAPIFile(c *gin.Context) {
	if s.openAPI == "" {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "OpenAPI 파일이 없습니다.", nil)
		return
	}
	c.File(s.openAPI)
}

func (s *Server) internal(c *gin.Context, err error) {
	log.Printf("rid=%s err=%v", c.GetString(requestIDKey), err)
	writeError(c, http.StatusInternalServerError, "INTERNAL", "서버 오류가 발생했습니다.", nil)
}

func (s *Server) Seed(ctx context.Context) error {
	hashed, err := auth.Hash("password123")
	if err != nil {
		return err
	}
	user, err := s.store.CreateUser(ctx, "mina@example.com", hashed, "mina")
	if err != nil && err != store.ErrConflict {
		return err
	}
	if err == store.ErrConflict {
		user, err = s.store.FindUserByEmail(ctx, "mina@example.com")
		if err != nil {
			return err
		}
	}
	n, err := s.store.CountPosts(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		_, err = s.store.CreatePost(ctx, user.ID, "첫 글", "안녕하세요", []string{"intro"})
	}
	return err
}

func (s *Server) ProcessOutbox(ctx context.Context) error {
	mail, err := s.store.NextPendingMail(ctx)
	if err == store.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.mailer.Send(mail.ToEmail, mail.Subject, mail.Body); err != nil {
		return err
	}
	return s.store.MarkMailSent(ctx, mail.ID)
}

func (s *Server) RunMailer(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.ProcessOutbox(ctx); err != nil {
				log.Printf("mailer: %v", err)
			}
		}
	}
}

func locateDir(explicit string, candidates ...string) string {
	if explicit != "" {
		if st, err := os.Stat(explicit); err == nil && st.IsDir() {
			return explicit
		}
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
	}
	return ""
}

func locateFile(explicit string, candidates ...string) string {
	if explicit != "" {
		if st, err := os.Stat(explicit); err == nil && !st.IsDir() {
			return explicit
		}
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return ""
}

const docsHTML = `<!doctype html>
<meta charset="utf-8">
<title>Gin 게시판 API 문서</title>
<h1>게시판 API</h1>
<p>OpenAPI 파일은 <a href="/api-docs/openapi.yaml">/api-docs/openapi.yaml</a> 입니다.</p>
<ul>
  <li>GET /health</li>
  <li>POST /api/auth/signup · POST /api/auth/login · GET /api/auth/me</li>
  <li>GET/POST /api/posts · GET/PATCH/DELETE /api/posts/{id}</li>
  <li>GET/POST /api/posts/{id}/comments · DELETE /api/comments/{id}</li>
</ul>
<p>인증은 Authorization: Bearer 액세스 토큰입니다. 화면 실습은 <a href="/">/</a> 입니다.</p>
`
