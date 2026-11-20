package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHello(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := newEngine()
	req := httptest.NewRequest(http.MethodGet, "/?name=<b>민아</b>", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "민아") || strings.Contains(rec.Body.String(), "<b>") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	ping := httptest.NewRequest(http.MethodGet, "/ping", nil)
	prec := httptest.NewRecorder()
	r.ServeHTTP(prec, ping)
	if !strings.Contains(prec.Body.String(), "pong") {
		t.Fatal(prec.Body.String())
	}
	if _, err := divide(1, 0); err == nil {
		t.Fatal("expected error")
	}
	if waitDone() != "완료" {
		t.Fatal("goroutine")
	}
}
