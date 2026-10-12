package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/auth"
)

func TestUserID_ContextWithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDKey, "user-1")
	uid, ok := UserID(ctx)
	if !ok || uid != "user-1" {
		t.Fatalf("got (%q,%v), want (user-1,true)", uid, ok)
	}
}

func TestUserID_ContextWithoutValue(t *testing.T) {
	uid, ok := UserID(context.Background())
	if ok || uid != "" {
		t.Fatalf("got (%q,%v), want ('',false)", uid, ok)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	tok, _ := auth.NewToken("secret", "user-1", time.Hour)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		uid, _ := UserID(r.Context())
		if uid != "user-1" {
			t.Errorf("context uid=%q, want user-1", uid)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	Auth("secret")(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
	if !called {
		t.Fatal("next not called")
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	Auth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")
	rr := httptest.NewRecorder()
	Auth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
}

func TestAuthMiddleware_WrongPrefix(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic something")
	rr := httptest.NewRecorder()
	Auth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
}
