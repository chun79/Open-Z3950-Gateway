package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIKeyAuth(t *testing.T) {
	os.Setenv("GATEWAY_API_KEY", "test-secret")
	defer os.Unsetenv("GATEWAY_API_KEY")

	r := gin.New()
	r.Use(authMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(200, "ok") })

	req, _ := http.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	req, _ = http.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-API-Key", "test-secret")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}