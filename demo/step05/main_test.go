package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindAndReplay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := newEngine()
	body := `{"title":"안녕","body":"내용"}`
	first := post(r, body, "k1")
	second := post(r, body, "k1")
	if first.Code != 201 || second.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("%d %d", first.Code, second.Code)
	}
	bad := post(r, `{"title":""}`, "")
	if bad.Code != 400 || !strings.Contains(bad.Body.String(), "VALIDATION_ERROR") {
		t.Fatal(bad.Body.String())
	}
}

func post(r http.Handler, body, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}
