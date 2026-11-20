package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samam0234/my-go-gin-study/internal/store"
)

type postIn struct {
	Title string   `json:"title" binding:"required,min=1,max=200"`
	Body  string   `json:"body" binding:"required,min=1,max=20000"`
	Tags  []string `json:"tags" binding:"omitempty,max=10,dive,max=30"`
}

type patchPostIn struct {
	Title *string   `json:"title" binding:"omitempty,min=1,max=200"`
	Body  *string   `json:"body" binding:"omitempty,min=1,max=20000"`
	Tags  *[]string `json:"tags" binding:"omitempty,max=10,dive,max=30"`
}

type commentIn struct {
	Body string `json:"body" binding:"required,min=1,max=2000"`
}

type listQuery struct {
	Q     string `form:"q"`
	Page  *int   `form:"page"`
	Limit *int   `form:"limit"`
	Sort  string `form:"sort"`
}

func pathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "id는 1 이상의 숫자여야 합니다.", nil)
		return 0, false
	}
	return id, true
}

func (s *Server) listPosts(c *gin.Context) {
	key := "posts?" + c.Request.URL.RawQuery
	if cached, ok := s.cache.Get(key, time.Now()); ok {
		c.Header("X-Cache", "HIT")
		writeRaw(c, http.StatusOK, cached)
		return
	}
	var query listQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "쿼리 파라미터를 확인하세요.", nil)
		return
	}
	page := 1
	if query.Page != nil {
		if *query.Page < 1 {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", map[string][]string{"page": {"1 이상이어야 합니다."}})
			return
		}
		page = *query.Page
	}
	limit := 10
	if query.Limit != nil {
		if *query.Limit < 1 {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", map[string][]string{"limit": {"1 이상이어야 합니다."}})
			return
		}
		limit = *query.Limit
		if limit > 50 {
			limit = 50
		}
	}
	sort := query.Sort
	if sort == "" {
		sort = "-createdAt"
	}
	switch sort {
	case "createdAt", "-createdAt", "-commentCount":
	default:
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", map[string][]string{"sort": {"허용된 정렬이 아닙니다."}})
		return
	}
	rows, total, err := s.store.ListPosts(c.Request.Context(), store.PostFilter{Q: query.Q, Page: page, Limit: limit, Sort: sort})
	if err != nil {
		s.internal(c, err)
		return
	}
	data := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		data = append(data, postListJSON(row))
	}
	pages := 0
	if total > 0 {
		pages = (total + limit - 1) / limit
	}
	body := listBytes(data, gin.H{
		"page": page, "limit": limit, "total": total, "totalPages": pages, "q": strings.TrimSpace(query.Q),
	})
	s.cache.Set(key, body, time.Now(), 5*time.Second)
	c.Header("X-Cache", "MISS")
	writeRaw(c, http.StatusOK, body)
}

func (s *Server) createPost(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if len(key) > 80 {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", map[string][]string{
			"Idempotency-Key": {"80자 이하로 입력하세요."},
		})
		return
	}
	if key != "" {
		if saved, hit := s.idem.get(user.ID, key, time.Now()); hit {
			if saved.location != "" {
				c.Header("Location", saved.location)
			}
			c.Header("Idempotency-Replayed", "true")
			writeRaw(c, saved.status, saved.body)
			return
		}
	}
	var in postIn
	if !bindJSON(c, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Body = strings.TrimSpace(in.Body)
	fields := map[string][]string{}
	if in.Title == "" {
		fields["title"] = []string{"1~200자로 입력하세요."}
	}
	if in.Body == "" {
		fields["body"] = []string{"내용을 입력하세요."}
	}
	if len(fields) > 0 {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", fields)
		return
	}
	post, err := s.store.CreatePost(c.Request.Context(), user.ID, in.Title, in.Body, in.Tags)
	if err != nil {
		s.internal(c, err)
		return
	}
	s.cache.DeletePrefix("posts?")
	body := dataBytes(postDetailJSON(post))
	location := fmt.Sprintf("/api/posts/%d", post.ID)
	if key != "" {
		s.idem.put(user.ID, key, http.StatusCreated, body, location, time.Now().Add(10*time.Minute))
	}
	c.Header("Location", location)
	note, _ := json.Marshal(gin.H{"type": "post.created", "data": postListJSON(post)})
	s.hub.Broadcast(note)
	writeRaw(c, http.StatusCreated, body)
}

func (s *Server) getPost(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	post, err := s.store.GetPost(c.Request.Context(), id)
	if err == store.ErrNotFound {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "자원을 찾을 수 없습니다.", nil)
		return
	}
	if err != nil {
		s.internal(c, err)
		return
	}
	writeData(c, http.StatusOK, postDetailJSON(post))
}

func (s *Server) patchPost(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	current, err := s.store.GetPost(c.Request.Context(), id)
	if err == store.ErrNotFound {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "자원을 찾을 수 없습니다.", nil)
		return
	}
	if err != nil {
		s.internal(c, err)
		return
	}
	if !canEdit(user, current.AuthorID) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "이 작업을 할 권한이 없습니다.", nil)
		return
	}
	var in patchPostIn
	if !bindJSON(c, &in) {
		return
	}
	title, body := current.Title, current.Body
	tags := decodeTags(current.Tags)
	if in.Title != nil {
		title = strings.TrimSpace(*in.Title)
	}
	if in.Body != nil {
		body = strings.TrimSpace(*in.Body)
	}
	if in.Tags != nil {
		tags = *in.Tags
	}
	if title == "" || body == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", map[string][]string{
			"title": {"1~200자로 입력하세요."},
		})
		return
	}
	updated, err := s.store.UpdatePost(c.Request.Context(), id, title, body, tags)
	if err != nil {
		s.internal(c, err)
		return
	}
	s.cache.DeletePrefix("posts?")
	writeData(c, http.StatusOK, postDetailJSON(updated))
}

func (s *Server) deletePost(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	current, err := s.store.GetPost(c.Request.Context(), id)
	if err == store.ErrNotFound {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "자원을 찾을 수 없습니다.", nil)
		return
	}
	if err != nil {
		s.internal(c, err)
		return
	}
	if !canEdit(user, current.AuthorID) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "이 작업을 할 권한이 없습니다.", nil)
		return
	}
	if err := s.store.DeletePost(c.Request.Context(), id); err != nil {
		s.internal(c, err)
		return
	}
	s.cache.DeletePrefix("posts?")
	noContent(c)
}

func (s *Server) listComments(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if _, err := s.store.GetPost(c.Request.Context(), id); err == store.ErrNotFound {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "자원을 찾을 수 없습니다.", nil)
		return
	} else if err != nil {
		s.internal(c, err)
		return
	}
	rows, err := s.store.ListComments(c.Request.Context(), id)
	if err != nil {
		s.internal(c, err)
		return
	}
	data := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		data = append(data, commentJSON(row))
	}
	writeData(c, http.StatusOK, data)
}

func (s *Server) createComment(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	if _, err := s.store.GetPost(c.Request.Context(), id); err == store.ErrNotFound {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "자원을 찾을 수 없습니다.", nil)
		return
	} else if err != nil {
		s.internal(c, err)
		return
	}
	var in commentIn
	if !bindJSON(c, &in) {
		return
	}
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", map[string][]string{
			"body": {"내용을 입력하세요."},
		})
		return
	}
	row, err := s.store.CreateComment(c.Request.Context(), id, user.ID, in.Body)
	if err != nil {
		s.internal(c, err)
		return
	}
	s.cache.DeletePrefix("posts?")
	c.Header("Location", fmt.Sprintf("/api/comments/%d", row.ID))
	writeData(c, http.StatusCreated, commentJSON(row))
}

func (s *Server) deleteComment(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	row, err := s.store.GetComment(c.Request.Context(), id)
	if err == store.ErrNotFound {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "자원을 찾을 수 없습니다.", nil)
		return
	}
	if err != nil {
		s.internal(c, err)
		return
	}
	if !canEdit(user, row.AuthorID) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "이 작업을 할 권한이 없습니다.", nil)
		return
	}
	if err := s.store.DeleteComment(c.Request.Context(), id); err != nil {
		s.internal(c, err)
		return
	}
	s.cache.DeletePrefix("posts?")
	noContent(c)
}
