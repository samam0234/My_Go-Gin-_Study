package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := newEngine()
	cases := []struct {
		path string
		code int
	}{
		{"/api/posts?q=" + url.QueryEscape("첫 글"), 200},
		{"/api/posts/2", 200},
		{"/api/posts/abc", 400},
		{"/nope", 404},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.code {
			t.Fatalf("%s %d", tc.path, rec.Code)
		}
	}
}
