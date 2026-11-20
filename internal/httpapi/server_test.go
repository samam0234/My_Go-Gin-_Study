package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/samam0234/my-go-gin-study/internal/config"
	"github.com/samam0234/my-go-gin-study/internal/httpapi"
	"github.com/samam0234/my-go-gin-study/internal/store"
)

func newServer(t *testing.T, tweak func(*config.Config)) (*httpapi.Server, *store.Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	dsn := "file:" + filepath.ToSlash(filepath.Join(dir, "t.db"))
	st, err := store.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Env:          "test",
		Driver:       "sqlite",
		DSN:          dsn,
		JWTSecret:    "test-secret-test-secret",
		JWTTTL:       time.Hour,
		CORSOrigin:   "http://localhost:5173",
		AIProvider:   "mock",
		AIModel:      "mock",
		AIDailyLimit: 100000,
		MCPToken:     "mcp-test-token",
		UploadDir:    filepath.Join(dir, "up"),
		MaxUpload:    2048,
		OAuthMock:    true,
	}
	if tweak != nil {
		tweak(&cfg)
	}
	return httpapi.New(cfg, st), st
}

func do(h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func asMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json %d %s: %v", rec.Code, rec.Body.String(), err)
	}
	return out
}

func tokenOf(t *testing.T, h http.Handler, email string) string {
	t.Helper()
	rec := do(h, http.MethodPost, "/api/auth/login", "", map[string]string{"email": email, "password": "password123"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login %d %s", rec.Code, rec.Body.String())
	}
	return asMap(t, rec)["data"].(map[string]any)["accessToken"].(string)
}

func signup(t *testing.T, h http.Handler, email, nickname string) {
	t.Helper()
	rec := do(h, http.MethodPost, "/api/auth/signup", "", map[string]string{
		"email": email, "password": "password123", "nickname": nickname,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup %s %d %s", email, rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "password123") || strings.Contains(rec.Body.String(), "$2") {
		t.Fatalf("password leaked: %s", rec.Body.String())
	}
}

func TestBoardFlow(t *testing.T) {
	srv, st := newServer(t, nil)
	h := srv.Router()

	health := do(h, http.MethodGet, "/health", "", nil)
	if health.Code != 200 || asMap(t, health)["status"] != "ok" {
		t.Fatalf("health %#v", health.Body.String())
	}
	if health.Header().Get("Access-Control-Allow-Origin") == "" {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
			t.Fatalf("cors %s", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	}

	signup(t, h, "mina@example.com", "mina")
	dup := do(h, http.MethodPost, "/api/auth/signup", "", map[string]string{
		"email": "mina@example.com", "password": "password123", "nickname": "mina",
	})
	if dup.Code != http.StatusConflict || asMap(t, dup)["error"].(map[string]any)["code"] != "EMAIL_TAKEN" {
		t.Fatalf("dup %#v", dup.Body.String())
	}
	badLogin := do(h, http.MethodPost, "/api/auth/login", "", map[string]string{"email": "none@example.com", "password": "password123"})
	wrong := do(h, http.MethodPost, "/api/auth/login", "", map[string]string{"email": "mina@example.com", "password": "wrongpass1"})
	if badLogin.Code != 401 || wrong.Code != 401 {
		t.Fatalf("login codes %d %d", badLogin.Code, wrong.Code)
	}
	if asMap(t, badLogin)["error"].(map[string]any)["code"] != asMap(t, wrong)["error"].(map[string]any)["code"] {
		t.Fatal("login errors should match")
	}

	token := tokenOf(t, h, "mina@example.com")
	if do(h, http.MethodGet, "/api/auth/me", "", nil).Code != 401 {
		t.Fatal("me should require auth")
	}
	me := do(h, http.MethodGet, "/api/auth/me", token, nil)
	if me.Code != 200 || asMap(t, me)["data"].(map[string]any)["nickname"] != "mina" {
		t.Fatalf("me %s", me.Body.String())
	}
	if do(h, http.MethodPost, "/api/auth/logout", token, nil).Code != 204 {
		t.Fatal("logout")
	}

	invalid := do(h, http.MethodPost, "/api/posts", token, map[string]any{"title": "", "body": "x"})
	if invalid.Code != 400 || asMap(t, invalid)["error"].(map[string]any)["code"] != "VALIDATION_ERROR" {
		t.Fatalf("validation %s", invalid.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/posts", strings.NewReader(`{"title":"첫 글","body":"안녕하세요","tags":["intro"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", "same-key")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 201 || rec.Header().Get("Location") != "/api/posts/1" {
		t.Fatalf("create %d %s loc=%s", rec.Code, rec.Body.String(), rec.Header().Get("Location"))
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/posts", strings.NewReader(`{"title":"첫 글","body":"안녕하세요","tags":["intro"]}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Idempotency-Key", "same-key")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != 201 || rec2.Header().Get("Idempotency-Replayed") != "true" || rec2.Body.String() != rec.Body.String() {
		t.Fatalf("replay %d %s", rec2.Code, rec2.Body.String())
	}

	list1 := do(h, http.MethodGet, "/api/posts?q=안녕", "", nil)
	if list1.Header().Get("X-Cache") != "MISS" {
		t.Fatalf("cache miss %s", list1.Header().Get("X-Cache"))
	}
	meta := asMap(t, list1)["meta"].(map[string]any)
	if meta["total"].(float64) != 1 || meta["q"] != "안녕" {
		t.Fatalf("meta %#v", meta)
	}
	list2 := do(h, http.MethodGet, "/api/posts?q=안녕", "", nil)
	if list2.Header().Get("X-Cache") != "HIT" {
		t.Fatal("expected cache hit")
	}
	if do(h, http.MethodGet, "/api/posts?sort=drop", "", nil).Code != 400 {
		t.Fatal("bad sort")
	}
	if do(h, http.MethodGet, "/api/posts/nope", "", nil).Code != 400 {
		t.Fatal("bad id")
	}
	clamped := do(h, http.MethodGet, "/api/posts?limit=100", "", nil)
	if asMap(t, clamped)["meta"].(map[string]any)["limit"].(float64) != 50 {
		t.Fatalf("limit %#v", asMap(t, clamped)["meta"])
	}

	signup(t, h, "kim@example.com", "kim")
	kim := tokenOf(t, h, "kim@example.com")
	if do(h, http.MethodPatch, "/api/posts/1", kim, map[string]string{"title": "훔침"}).Code != 403 {
		t.Fatal("expected forbidden")
	}
	if do(h, http.MethodPatch, "/api/posts/1", token, map[string]string{"title": "수정한 글"}).Code != 200 {
		t.Fatal("patch")
	}
	comment := do(h, http.MethodPost, "/api/posts/1/comments", kim, map[string]string{"body": "좋아요"})
	if comment.Code != 201 {
		t.Fatalf("comment %s", comment.Body.String())
	}
	if do(h, http.MethodGet, "/api/posts/1/comments", "", nil).Code != 200 {
		t.Fatal("comments")
	}
	if err := st.SetRole(t.Context(), 2, "admin"); err != nil {
		t.Fatal(err)
	}
	kim = tokenOf(t, h, "kim@example.com")
	if do(h, http.MethodDelete, "/api/comments/1", kim, nil).Code != 204 {
		t.Fatal("admin comment delete")
	}
	again := do(h, http.MethodPost, "/api/posts/1/comments", token, map[string]string{"body": "다시"})
	if again.Code != 201 {
		t.Fatal(again.Body.String())
	}
	if do(h, http.MethodDelete, "/api/posts/1", token, nil).Code != 204 {
		t.Fatal("delete post")
	}
	if do(h, http.MethodDelete, "/api/comments/2", token, nil).Code != 404 {
		t.Fatal("comment should cascade")
	}
	missing := do(h, http.MethodGet, "/api/missing", "", nil)
	if missing.Code != 404 || asMap(t, missing)["error"].(map[string]any)["code"] != "NOT_FOUND" {
		t.Fatalf("404 %s", missing.Body.String())
	}
}

func TestDocumentAIAndMCP(t *testing.T) {
	srv, _ := newServer(t, nil)
	h := srv.Router()
	signup(t, h, "mina@example.com", "mina")
	token := tokenOf(t, h, "mina@example.com")
	doc := do(h, http.MethodPost, "/api/documents", token, map[string]string{
		"title":   "Gin 소개",
		"content": "Gin은 Go 언어의 HTTP 웹 프레임워크다.\n라우터와 미들웨어를 제공한다.",
	})
	if doc.Code != 201 {
		t.Fatalf("doc %s", doc.Body.String())
	}
	ask := do(h, http.MethodPost, "/api/documents/1/questions", token, map[string]string{"question": "Gin은 무엇인가"})
	if ask.Code != 201 || !strings.Contains(ask.Body.String(), "발췌") {
		t.Fatalf("ask %s", ask.Body.String())
	}
	chat := do(h, http.MethodPost, "/api/ai/chat", token, map[string]any{
		"messages": []map[string]string{{"role": "user", "content": "글을 짧게 다듬어 줘"}},
	})
	if chat.Code != 200 || !strings.Contains(chat.Body.String(), "다듬으면") {
		t.Fatalf("chat %s", chat.Body.String())
	}
	stream := do(h, http.MethodPost, "/api/ai/chat/stream", token, map[string]any{
		"messages": []map[string]string{{"role": "user", "content": "안녕"}},
	})
	if stream.Code != 200 || !strings.Contains(stream.Body.String(), "data:") || !strings.Contains(stream.Body.String(), "[DONE]") {
		t.Fatalf("stream %s", stream.Body.String())
	}

	initReq := do(h, http.MethodPost, "/mcp", "", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize"})
	if initReq.Code != 401 {
		t.Fatal("mcp should require token")
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	req.Header.Set("Authorization", "Bearer mcp-test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "search_posts") {
		t.Fatalf("tools %s", rec.Body.String())
	}
}

func TestUploadPaymentOAuthQuota(t *testing.T) {
	srv, st := newServer(t, func(cfg *config.Config) { cfg.AIDailyLimit = 1 })
	h := srv.Router()
	signup(t, h, "mina@example.com", "mina")
	token := tokenOf(t, h, "mina@example.com")
	if err := st.AddUsage(t.Context(), 1, 5, 5); err != nil {
		t.Fatal(err)
	}
	quota := do(h, http.MethodPost, "/api/ai/chat", token, map[string]any{
		"messages": []map[string]string{{"role": "user", "content": "안녕"}},
	})
	if quota.Code != 429 || asMap(t, quota)["error"].(map[string]any)["code"] != "AI_QUOTA_EXCEEDED" {
		t.Fatalf("quota %s", quota.Body.String())
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "note.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("hello upload"))
	_ = writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/uploads", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("upload %s", rec.Body.String())
	}
	url := asMap(t, rec)["data"].(map[string]any)["url"].(string)
	if do(h, http.MethodGet, url, "", nil).Code != 200 {
		t.Fatal("download")
	}

	pay := do(h, http.MethodPost, "/api/payments", token, map[string]any{"orderId": "order-1", "amount": 1200})
	if pay.Code != 201 {
		t.Fatalf("pay %s", pay.Body.String())
	}
	if do(h, http.MethodPost, "/api/payments/1/confirm", token, nil).Code != 200 {
		t.Fatal("confirm")
	}
	mail := do(h, http.MethodPost, "/api/mail/test", token, map[string]string{})
	if mail.Code != 202 {
		t.Fatalf("mail %s", mail.Body.String())
	}
	start := do(h, http.MethodGet, "/api/auth/oauth/google/start", "", nil)
	authURL := asMap(t, start)["data"].(map[string]any)["authUrl"].(string)
	callback := do(h, http.MethodGet, authURL, "", nil)
	if callback.Code != 200 || asMap(t, callback)["data"].(map[string]any)["accessToken"] == "" {
		t.Fatalf("oauth %s", callback.Body.String())
	}
}

func TestRateLimitAndSocket(t *testing.T) {
	srv, _ := newServer(t, func(cfg *config.Config) { cfg.RatePerMin = 2 })
	h := srv.Router()
	if do(h, http.MethodGet, "/health", "", nil).Code != 200 {
		t.Fatal("health is not limited")
	}
	if do(h, http.MethodGet, "/api/posts", "", nil).Code != 200 {
		t.Fatal("first")
	}
	if do(h, http.MethodGet, "/api/posts", "", nil).Code != 200 {
		t.Fatal("second")
	}
	limited := do(h, http.MethodGet, "/api/posts", "", nil)
	if limited.Code != 429 {
		t.Fatalf("rate %d", limited.Code)
	}

	openSrv, _ := newServer(t, nil)
	signup(t, openSrv.Router(), "mina@example.com", "mina")
	token := tokenOf(t, openSrv.Router(), "mina@example.com")
	ts := httptest.NewServer(openSrv.Router())
	defer ts.Close()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?access_token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	made := do(openSrv.Router(), http.MethodPost, "/api/posts", token, map[string]string{"title": "알림", "body": "실시간"})
	if made.Code != 201 {
		t.Fatal(made.Body.String())
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil || !strings.Contains(string(message), "post.created") {
		t.Fatalf("ws %v %s", err, message)
	}
}
