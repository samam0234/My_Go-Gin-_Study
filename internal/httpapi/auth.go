package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samam0234/my-go-gin-study/internal/auth"
	"github.com/samam0234/my-go-gin-study/internal/store"
)

type signupIn struct {
	Email    string `json:"email" binding:"required,email,max=200"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Nickname string `json:"nickname" binding:"required,min=1,max=40"`
}

type loginIn struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) requireAuth(c *gin.Context) {
	header := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) || strings.TrimSpace(header[len(prefix):]) == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	id, err := auth.Parse(s.cfg.JWTSecret, strings.TrimSpace(header[len(prefix):]))
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "토큰이 유효하지 않습니다.", nil)
		return
	}
	user, err := s.store.FindUserByID(c.Request.Context(), id)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "토큰이 유효하지 않습니다.", nil)
		return
	}
	c.Set(userKey, user)
	c.Next()
}

func currentUser(c *gin.Context) (store.User, bool) {
	value, ok := c.Get(userKey)
	if !ok {
		return store.User{}, false
	}
	user, ok := value.(store.User)
	return user, ok
}

func (s *Server) signup(c *gin.Context) {
	var in signupIn
	if !bindJSON(c, &in) {
		return
	}
	in.Nickname = strings.TrimSpace(in.Nickname)
	if in.Nickname == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", map[string][]string{
			"nickname": {"1~40자로 입력하세요."},
		})
		return
	}
	hashed, err := auth.Hash(in.Password)
	if err != nil {
		s.internal(c, err)
		return
	}
	user, err := s.store.CreateUser(c.Request.Context(), in.Email, hashed, in.Nickname)
	if err == store.ErrConflict {
		writeError(c, http.StatusConflict, "EMAIL_TAKEN", "이미 가입된 이메일입니다.", nil)
		return
	}
	if err != nil {
		s.internal(c, err)
		return
	}
	if err := s.store.EnqueueMail(c.Request.Context(), user.Email, "가입을 환영합니다", user.Nickname+"님, 게시판에 오신 것을 환영합니다."); err != nil {
		s.internal(c, err)
		return
	}
	writeData(c, http.StatusCreated, publicUser(user))
}

func (s *Server) login(c *gin.Context) {
	var in loginIn
	if !bindJSON(c, &in) {
		return
	}
	user, err := s.store.FindUserByEmail(c.Request.Context(), in.Email)
	if err != nil && err != store.ErrNotFound {
		s.internal(c, err)
		return
	}
	hash := ""
	if err == nil {
		hash = user.PasswordHash
	}
	if !auth.Check(hash, in.Password) {
		writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "이메일 또는 비밀번호를 확인하세요.", nil)
		return
	}
	token, err := auth.Sign(s.cfg.JWTSecret, user.ID, s.cfg.JWTTTL, time.Now())
	if err != nil {
		s.internal(c, err)
		return
	}
	writeData(c, http.StatusOK, gin.H{"user": publicUser(user), "accessToken": token})
}

func (s *Server) logout(c *gin.Context) {
	noContent(c)
}

func (s *Server) me(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	writeData(c, http.StatusOK, publicUser(user))
}

func (s *Server) issueToken(c *gin.Context, user store.User) {
	token, err := auth.Sign(s.cfg.JWTSecret, user.ID, s.cfg.JWTTTL, time.Now())
	if err != nil {
		s.internal(c, err)
		return
	}
	if strings.Contains(c.GetHeader("Accept"), "text/html") {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, `<!doctype html><meta charset="utf-8"><title>로그인</title><p>로그인되었습니다.</p><script>localStorage.setItem("accessToken", %q); location.replace("/");</script>`, token)
		return
	}
	writeData(c, http.StatusOK, gin.H{"user": publicUser(user), "accessToken": token})
}

func secureEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
