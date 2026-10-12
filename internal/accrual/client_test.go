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

func TestClient_GetOrder_504(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetOrder(context.Background(), "123")
	if err == nil {
		t.Fatal("expected error for 504")
	}
}

func TestTooManyRequestsError_Error(t *testing.T) {
	e := &TooManyRequestsError{RetryAfter: 42 * time.Second}
	want := "too many requests to accrual system"
	if e.Error() != want {
		t.Fatalf("Error() = %q, want %q", e.Error(), want)
	}
}

func TestTooManyRequestsError_Is(t *testing.T) {
	e := &TooManyRequestsError{RetryAfter: 0}
	if !e.Is(ErrTooManyRequests) {
		t.Fatal("Is(ErrTooManyRequests) should be true")
	}
	other := errors.New("other")
	if e.Is(other) {
		t.Fatal("Is(other) should be false")
	}
}

func TestParseRetryAfter_Empty(t *testing.T) {
	d := parseRetryAfter("")
	if d != time.Minute {
		t.Fatalf("want 1m, got %v", d)
	}
}

func TestParseRetryAfter_Numeric(t *testing.T) {
	d := parseRetryAfter("120")
	if d != 120*time.Second {
		t.Fatalf("want 120s, got %v", d)
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	expected := time.Now().UTC().Add(45 * time.Second)
	h := expected.Format(http.TimeFormat)
	d := parseRetryAfter(h)
	if d < 42*time.Second || d > 48*time.Second {
		t.Fatalf("want ~45s, got %v", d)
	}
}

func TestParseRetryAfter_InvalidFallback(t *testing.T) {
	d := parseRetryAfter("not-a-date")
	if d != time.Minute {
		t.Fatalf("want 1m fallback, got %v", d)
	}
}
