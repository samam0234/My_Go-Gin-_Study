package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPackages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := newEngine()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Fatal(rec.Body.String())
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/memos", nil))
	if !strings.Contains(rec.Body.String(), "패키지") {
		t.Fatal(rec.Body.String())
	}
}
