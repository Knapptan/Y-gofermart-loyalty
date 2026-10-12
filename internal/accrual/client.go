package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Внешние статусы сервиса расчёта.
const (
	StatusRegistered = "REGISTERED"
	StatusProcessing = "PROCESSING"
	StatusInvalid    = "INVALID"
	StatusProcessed  = "PROCESSED"
)

var (
	ErrNotRegistered   = errors.New("order not registered in accrual system")
	ErrTooManyRequests = errors.New("too many requests to accrual system")
)

// Response — успешный ответ сервиса (200).
type Response struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

// TooManyRequestsError несёт Retry-After из заголовка.
type TooManyRequestsError struct {
	RetryAfter time.Duration
}

func (e *TooManyRequestsError) Error() string {
	return "too many requests to accrual system"
}

func (e *TooManyRequestsError) Is(target error) bool {
	return target == ErrTooManyRequests
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// GetOrder возвращает:
//   - (*Response, nil)             при 200
//   - (nil, ErrNotRegistered)      при 204
//   - (nil, *TooManyRequestsError) при 429
//   - (nil, err)                   при 5xx и прочих
func (c *Client) GetOrder(ctx context.Context, number string) (*Response, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var r Response
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return nil, err
		}
		return &r, nil
	case http.StatusNoContent:
		return nil, ErrNotRegistered
	case http.StatusTooManyRequests:
		return nil, &TooManyRequestsError{
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	default:
		return nil, fmt.Errorf("accrual service returned %d", resp.StatusCode)
	}
}

func parseRetryAfter(h string) time.Duration {
	if h == "" {
		return time.Minute
	}
	if secs, err := strconv.Atoi(h); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		return time.Until(t)
	}
	return time.Minute
}
