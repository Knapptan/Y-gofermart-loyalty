package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/service"
)

// fakeUserRepo — in-memory заглушка.
type fakeUserRepo struct {
	byLogin map[string]*model.User
	nextID  int
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byLogin: map[string]*model.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, login, hash string) (*model.User, error) {
	if _, ok := f.byLogin[login]; ok {
		return nil, repository.ErrLoginExists
	}
	f.nextID++
	id := "user-" + login
	u := &model.User{ID: id, Login: login, PasswordHash: hash}
	f.byLogin[login] = u
	return u, nil
}

func (f *fakeUserRepo) GetByLogin(_ context.Context, login string) (*model.User, error) {
	u, ok := f.byLogin[login]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func newTestRouter() http.Handler {
	svc := service.NewUserService(newFakeUserRepo(), "test-secret")
	h := NewUserHandler(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user/register", h.Register)
	mux.HandleFunc("/api/user/login", h.Login)
	return mux
}

func doJSON(t *testing.T, h http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestRegister_Success(t *testing.T) {
	h := newTestRouter()
	rr := doJSON(t, h, "/api/user/register", map[string]string{
		"login": "alice", "password": "secret",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if tok := rr.Header().Get("Authorization"); tok == "" {
		t.Fatal("expected Authorization header with token")
	}
}

func TestRegister_Duplicate(t *testing.T) {
	h := newTestRouter()
	doJSON(t, h, "/api/user/register", map[string]string{"login": "a", "password": "b"})
	rr := doJSON(t, h, "/api/user/register", map[string]string{"login": "a", "password": "c"})
	if rr.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rr.Code)
	}
}

func TestRegister_EmptyCredentials(t *testing.T) {
	h := newTestRouter()
	rr := doJSON(t, h, "/api/user/register", map[string]string{"login": "", "password": ""})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestRegister_InvalidJSON(t *testing.T) {
	h := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader([]byte("not-json")))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	h := newTestRouter()
	doJSON(t, h, "/api/user/register", map[string]string{"login": "bob", "password": "pw"})
	rr := doJSON(t, h, "/api/user/login", map[string]string{"login": "bob", "password": "pw"})
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if tok := rr.Header().Get("Authorization"); tok == "" {
		t.Fatal("expected Authorization header")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	h := newTestRouter()
	doJSON(t, h, "/api/user/register", map[string]string{"login": "eve", "password": "right"})
	rr := doJSON(t, h, "/api/user/login", map[string]string{"login": "eve", "password": "wrong"})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	h := newTestRouter()
	rr := doJSON(t, h, "/api/user/login", map[string]string{"login": "ghost", "password": "x"})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}
