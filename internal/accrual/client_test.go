package accrual

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_GetOrder_200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order":"123","status":"PROCESSED","accrual":500}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	r, err := c.GetOrder(context.Background(), "123")
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusProcessed || r.Accrual != 500 {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestClient_GetOrder_204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetOrder(context.Background(), "123")
	if !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("want ErrNotRegistered, got %v", err)
	}
}

func TestClient_GetOrder_429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetOrder(context.Background(), "123")
	if !errors.Is(err, ErrTooManyRequests) {
		t.Fatalf("want ErrTooManyRequests, got %v", err)
	}
	var tooMany *TooManyRequestsError
	if !errors.As(err, &tooMany) {
		t.Fatal("expected *TooManyRequestsError")
	}
	if tooMany.RetryAfter != 30*time.Second {
		t.Fatalf("want 30s, got %v", tooMany.RetryAfter)
	}
}

func TestClient_GetOrder_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetOrder(context.Background(), "123")
	if err == nil {
		t.Fatal("expected error")
	}
}
