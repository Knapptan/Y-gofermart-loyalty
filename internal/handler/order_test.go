package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/auth"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/middleware"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/service"
)

const testSecret = "test-secret"

type fakeOrderRepo struct {
	orders map[string]*model.Order // number → order
}

func newFakeOrderRepo() *fakeOrderRepo {
	return &fakeOrderRepo{orders: map[string]*model.Order{}}
}

func (f *fakeOrderRepo) Create(_ context.Context, userID, number string) error {
	if o, ok := f.orders[number]; ok {
		if o.UserID == userID {
			return repository.ErrOrderExistsSame
		}
		return repository.ErrOrderExistsOther
	}
	f.orders[number] = &model.Order{
		Number: number, UserID: userID, Status: model.StatusNew,
	}
	return nil
}

func (f *fakeOrderRepo) GetByNumber(_ context.Context, number string) (*model.Order, error) {
	o, ok := f.orders[number]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return o, nil
}

func (f *fakeOrderRepo) ListByUser(_ context.Context, userID string) ([]model.Order, error) {
	var out []model.Order
	for _, o := range f.orders {
		if o.UserID == userID {
			out = append(out, *o)
		}
	}
	return out, nil
}

func (f *fakeOrderRepo) ListPending(_ context.Context) ([]model.Order, error) {
	return nil, nil
}

func (f *fakeOrderRepo) UpdateStatus(_ context.Context, _ string, _ model.OrderStatus, _ float64) error {
	return nil
}

func newOrderRouter(repo repository.OrderRepository) http.Handler {
	svc := service.NewOrderService(repo)
	h := NewOrderHandler(svc)
	mux := http.NewServeMux()
	// оборачиваем хендлеры в auth middleware
	mw := middleware.Auth(testSecret)
	mux.Handle("/orders", mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.Create(w, r)
		case http.MethodGet:
			h.List(w, r)
		}
	})))
	return mux
}

func tokenFor(t *testing.T, userID string) string {
	t.Helper()
	tok, err := auth.NewToken(testSecret, userID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func postOrder(t *testing.T, h http.Handler, token, number string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(number)))
	req.Header.Set("Content-Type", "text/plain")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestOrderCreate_NewOrder(t *testing.T) {
	h := newOrderRouter(newFakeOrderRepo())
	rr := postOrder(t, h, tokenFor(t, "u1"), "12345678903")
	if rr.Code != http.StatusAccepted {
		t.Fatalf("want 202, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestOrderCreate_InvalidLuhn(t *testing.T) {
	h := newOrderRouter(newFakeOrderRepo())
	rr := postOrder(t, h, tokenFor(t, "u1"), "123")
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", rr.Code)
	}
}

func TestOrderCreate_DuplicateSameUser(t *testing.T) {
	repo := newFakeOrderRepo()
	h := newOrderRouter(repo)
	tok := tokenFor(t, "u1")
	postOrder(t, h, tok, "12345678903")
	rr := postOrder(t, h, tok, "12345678903")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestOrderCreate_DuplicateOtherUser(t *testing.T) {
	h := newOrderRouter(newFakeOrderRepo())
	postOrder(t, h, tokenFor(t, "u1"), "12345678903")
	rr := postOrder(t, h, tokenFor(t, "u2"), "12345678903")
	if rr.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rr.Code)
	}
}

func TestOrderCreate_Unauthorized(t *testing.T) {
	h := newOrderRouter(newFakeOrderRepo())
	rr := postOrder(t, h, "", "12345678903")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestOrderList_Empty(t *testing.T) {
	h := newOrderRouter(newFakeOrderRepo())
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rr.Code)
	}
}

func TestOrderList_WithData(t *testing.T) {
	repo := newFakeOrderRepo()
	h := newOrderRouter(repo)
	tok := tokenFor(t, "u1")
	postOrder(t, h, tok, "12345678903")

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var got []model.Order
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if len(got) != 1 || got[0].Number != "12345678903" {
		t.Fatalf("unexpected list: %+v", got)
	}
}
