package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/open-z3950-gateway/pkg/auth"
	"github.com/yourusername/open-z3950-gateway/pkg/provider"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
	"golang.org/x/crypto/bcrypt"
)

func TestSearchHandler(t *testing.T) {
	mockProv := &MockProvider{}
	mockProv.SearchFunc = func(db string, query z3950.StructuredQuery) ([]string, error) {
		return []string{"1"}, nil
	}
	mockProv.FetchFunc = func(db string, ids []string) ([]*z3950.MARCRecord, error) {
		// Create a dummy record
		rec := z3950.BuildMARC(&z3950.ProfileMARC21, "1", "Go Book", "Author", "123", "Pub", "2024", "", "")
		parsed, _ := z3950.ParseMARC(rec)
		return []*z3950.MARCRecord{parsed}, nil
	}

	// Create Auth User
	mockProv.GetUserByUsernameFunc = func(username string) (*provider.User, error) {
		return &provider.User{Username: "test", Role: "user"}, nil
	}

	// MUST set environment variable BEFORE setupRouter because authMiddleware captures it
	t.Setenv("GATEWAY_API_KEY", "test-key")
	r := setupRouter(mockProv)

	// Bypass auth with API Key for simplicity in this test
	req, _ := http.NewRequest("GET", "/api/search?term1=Go&attr1=4", nil)
	req.Header.Set("X-API-Key", "test-key")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestAuthLogin(t *testing.T) {
	mockProv := &MockProvider{}
	hash, _ := auth.HashPassword("secret")

	mockProv.GetUserByUsernameFunc = func(username string) (*provider.User, error) {
		if username == "valid" {
			return &provider.User{Username: "valid", PasswordHash: hash, Role: "user"}, nil
		}
		return nil, bcrypt.ErrMismatchedHashAndPassword // Simulate not found
	}

	r := setupRouter(mockProv)

	// 1. Success
	body := `{"username": "valid", "password": "secret"}`
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Login failed. Code: %d, Body: %s", w.Code, w.Body.String())
	}

	// 2. Fail
	body = `{"username": "valid", "password": "wrong"}`
	req, _ = http.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestILLCreate(t *testing.T) {
	mockProv := &MockProvider{}
	mockProv.CreateILLRequestFunc = func(req provider.ILLRequest) error {
		return nil
	}

	t.Setenv("GATEWAY_API_KEY", "test")
	r := setupRouter(mockProv)

	illReq := provider.ILLRequest{Title: "Book", ISBN: "123"}
	jsonBytes, _ := json.Marshal(illReq)

	req, _ := http.NewRequest("POST", "/api/ill-requests", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d. Body: %s", w.Code, w.Body.String())
	}
}
