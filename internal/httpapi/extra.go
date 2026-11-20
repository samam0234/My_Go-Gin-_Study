package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samam0234/my-go-gin-study/internal/auth"
	"github.com/samam0234/my-go-gin-study/internal/mcp"
	"github.com/samam0234/my-go-gin-study/internal/store"
)

type payIn struct {
	OrderID string `json:"orderId" binding:"required,min=1,max=80"`
	Amount  int64  `json:"amount" binding:"required,min=1,max=10000000"`
}

type mailIn struct {
	To string `json:"to" binding:"omitempty,email"`
}

func (s *Server) oauthStart(c *gin.Context) {
	if s.cfg.OAuthMock {
		writeData(c, http.StatusOK, gin.H{"authUrl": "/api/auth/oauth/google/callback?code=demo-code&state=demo"})
		return
	}
	if s.cfg.GoogleClientID == "" || s.cfg.GoogleClientSecret == "" {
		writeError(c, http.StatusNotImplemented, "OAUTH_NOT_CONFIGURED", "Google OAuth 환경 변수가 없습니다.", nil)
		return
	}
	verifier := randomRaw(32)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	state := randomRaw(18)
	s.oauthMu.Lock()
	s.oauth[state] = oauthPending{verifier: verifier, exp: time.Now().Add(10 * time.Minute)}
	s.oauthMu.Unlock()
	authURL, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	query := authURL.Query()
	query.Set("client_id", s.cfg.GoogleClientID)
	query.Set("redirect_uri", s.cfg.GoogleRedirect)
	query.Set("response_type", "code")
	query.Set("scope", "openid email profile")
	query.Set("state", state)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	authURL.RawQuery = query.Encode()
	writeData(c, http.StatusOK, gin.H{"authUrl": authURL.String()})
}

func (s *Server) oauthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if s.cfg.OAuthMock {
		if code != "demo-code" || state != "demo" {
			writeError(c, http.StatusBadRequest, "BAD_REQUEST", "연습용 코드가 아닙니다.", nil)
			return
		}
		user, err := s.store.FindUserByEmail(c.Request.Context(), "oauth.mina@example.com")
		if err == store.ErrNotFound {
			hashed, hashErr := auth.Hash(randomRaw(24))
			if hashErr != nil {
				s.internal(c, hashErr)
				return
			}
			user, err = s.store.CreateUser(c.Request.Context(), "oauth.mina@example.com", hashed, "oauth-mina")
		}
		if err != nil {
			s.internal(c, err)
			return
		}
		s.issueToken(c, user)
		return
	}
	s.oauthMu.Lock()
	pending, ok := s.oauth[state]
	if ok && time.Now().After(pending.exp) {
		ok = false
	}
	delete(s.oauth, state)
	s.oauthMu.Unlock()
	if !ok || code == "" {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "로그인 상태가 만료되었습니다. 다시 시작하세요.", nil)
		return
	}
	profile, err := s.exchangeGoogle(c.Request.Context(), code, pending.verifier)
	if err != nil {
		log.Printf("rid=%s oauth: %v", c.GetString(requestIDKey), err)
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "소셜 로그인에 실패했습니다.", nil)
		return
	}
	user, err := s.store.FindUserByEmail(c.Request.Context(), profile.Email)
	if err == store.ErrNotFound {
		hashed, hashErr := auth.Hash(randomRaw(24))
		if hashErr != nil {
			s.internal(c, hashErr)
			return
		}
		nickname := profile.Nickname
		if nickname == "" {
			nickname = strings.Split(profile.Email, "@")[0]
		}
		runes := []rune(nickname)
		if len(runes) > 40 {
			nickname = string(runes[:40])
		}
		user, err = s.store.CreateUser(c.Request.Context(), profile.Email, hashed, nickname)
	}
	if err != nil {
		s.internal(c, err)
		return
	}
	s.issueToken(c, user)
}

func (s *Server) exchangeGoogleHTTP(ctx context.Context, code, verifier string) (googleProfile, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", s.cfg.GoogleClientID)
	form.Set("client_secret", s.cfg.GoogleClientSecret)
	form.Set("redirect_uri", s.cfg.GoogleRedirect)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", verifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return googleProfile{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return googleProfile{}, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 400 {
		return googleProfile{}, fmt.Errorf("google token %d", res.StatusCode)
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &token); err != nil || token.AccessToken == "" {
		return googleProfile{}, fmt.Errorf("google token parse")
	}
	infoReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return googleProfile{}, err
	}
	infoReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	infoRes, err := http.DefaultClient.Do(infoReq)
	if err != nil {
		return googleProfile{}, err
	}
	defer infoRes.Body.Close()
	infoRaw, _ := io.ReadAll(io.LimitReader(infoRes.Body, 1<<20))
	if infoRes.StatusCode >= 400 {
		return googleProfile{}, fmt.Errorf("google userinfo %d", infoRes.StatusCode)
	}
	var info struct {
		Email         string `json:"email"`
		Name          string `json:"name"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := json.Unmarshal(infoRaw, &info); err != nil {
		return googleProfile{}, err
	}
	if !info.EmailVerified || info.Email == "" {
		return googleProfile{}, fmt.Errorf("확인되지 않은 이메일")
	}
	return googleProfile{Email: info.Email, Nickname: info.Name}, nil
}

func (s *Server) upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, s.cfg.MaxUpload+4096)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if isTooLarge(err) {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "파일이 너무 큽니다.", nil)
			return
		}
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "file 필드가 필요합니다.", nil)
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".txt", ".md", ".png", ".jpg", ".jpeg", ".webp", ".pdf":
	default:
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", map[string][]string{
			"file": {"txt, md, png, jpg, webp, pdf 만 올릴 수 있습니다."},
		})
		return
	}
	if err := os.MkdirAll(s.cfg.UploadDir, 0o755); err != nil {
		s.internal(c, err)
		return
	}
	name := randomRaw(16) + ext
	path := filepath.Join(s.cfg.UploadDir, name)
	dst, err := os.Create(path)
	if err != nil {
		s.internal(c, err)
		return
	}
	defer dst.Close()
	n, err := io.Copy(dst, io.LimitReader(file, s.cfg.MaxUpload+1))
	if err != nil {
		_ = os.Remove(path)
		if isTooLarge(err) {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "파일이 너무 큽니다.", nil)
			return
		}
		s.internal(c, err)
		return
	}
	if n > s.cfg.MaxUpload {
		_ = os.Remove(path)
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "파일이 너무 큽니다.", nil)
		return
	}
	writeData(c, http.StatusCreated, gin.H{"filename": name, "size": n, "url": "/uploads/" + name})
}

func (s *Server) download(c *gin.Context) {
	name := c.Param("name")
	if name != filepath.Base(name) || strings.Contains(name, "..") {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "파일 이름이 올바르지 않습니다.", nil)
		return
	}
	path := filepath.Join(s.cfg.UploadDir, name)
	if _, err := os.Stat(path); err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "자원을 찾을 수 없습니다.", nil)
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.FileAttachment(path, name)
}

func (s *Server) createPayment(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	var in payIn
	if !bindJSON(c, &in) {
		return
	}
	row, err := s.store.CreatePayment(c.Request.Context(), user.ID, strings.TrimSpace(in.OrderID), in.Amount)
	if err == store.ErrConflict {
		existing, findErr := s.store.FindPaymentByOrder(c.Request.Context(), user.ID, strings.TrimSpace(in.OrderID))
		if findErr != nil {
			s.internal(c, findErr)
			return
		}
		if existing.Amount != in.Amount {
			writeError(c, http.StatusConflict, "CONFLICT", "같은 주문 번호가 다른 금액으로 있습니다.", nil)
			return
		}
		writeData(c, http.StatusOK, paymentJSON(existing))
		return
	}
	if err != nil {
		s.internal(c, err)
		return
	}
	writeData(c, http.StatusCreated, paymentJSON(row))
}

func (s *Server) confirmPayment(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	row, err := s.store.ConfirmPayment(c.Request.Context(), id, user.ID)
	if err == store.ErrNotFound {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "자원을 찾을 수 없습니다.", nil)
		return
	}
	if err != nil {
		s.internal(c, err)
		return
	}
	writeData(c, http.StatusOK, paymentJSON(row))
}

func (s *Server) mailTest(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	var in mailIn
	if c.Request.ContentLength != 0 {
		if !bindJSON(c, &in) {
			return
		}
	}
	to := strings.TrimSpace(in.To)
	if to == "" {
		to = user.Email
	}
	if err := s.store.EnqueueMail(c.Request.Context(), to, "테스트 메일", "학습용 메일 대기열에 넣었습니다."); err != nil {
		s.internal(c, err)
		return
	}
	if err := s.ProcessOutbox(c.Request.Context()); err != nil {
		s.internal(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"status": "sent", "to": to}})
}

func (s *Server) socket(c *gin.Context) {
	token := c.Query("access_token")
	if token == "" {
		header := c.GetHeader("Authorization")
		token = strings.TrimPrefix(header, "Bearer ")
		if token == header {
			token = ""
		}
	}
	if _, err := auth.Parse(s.cfg.JWTSecret, token); err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := s.hub.Add(conn)
	defer s.hub.Remove(client)
	conn.SetReadLimit(1024)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (s *Server) allowOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range splitOrigins(s.cfg.CORSOrigin) {
		if origin == allowed {
			return true
		}
	}
	if s.cfg.Env == "production" {
		return false
	}
	return strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")
}

func (s *Server) mcpHTTP(c *gin.Context) {
	header := c.GetHeader("Authorization")
	token := strings.TrimPrefix(header, "Bearer ")
	if token == header || !secureEqual(token, s.cfg.MCPToken) {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "MCP 토큰이 필요합니다.", nil)
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		s.internal(c, err)
		return
	}
	resp, err := mcp.Handle(c.Request.Context(), s.store, body)
	if err != nil {
		s.internal(c, err)
		return
	}
	if resp == nil {
		c.Status(http.StatusAccepted)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", resp)
}

func randomRaw(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
