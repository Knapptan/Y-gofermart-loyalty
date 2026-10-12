package service

import (
	"testing"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

func TestIsOrderConflict_Same(t *testing.T) {
	same, _ := IsOrderConflict(repository.ErrOrderExistsSame)
	if !same {
		t.Fatal("expected same=true for ErrOrderExistsSame")
	}
}

func TestIsOrderConflict_Other(t *testing.T) {
	_, other := IsOrderConflict(repository.ErrOrderExistsOther)
	if !other {
		t.Fatal("expected other=true for ErrOrderExistsOther")
	}
}

func TestIsOrderConflict_Nil(t *testing.T) {
	same, other := IsOrderConflict(nil)
	if same || other {
		t.Fatal("expected both false for nil")
	}
}

type wrappedErr struct{ error }

func (e *wrappedErr) Unwrap() error { return e.error }

func TestIsOrderConflict_Wrapped(t *testing.T) {
	wrapped := &wrappedErr{repository.ErrOrderExistsSame}
	same, _ := IsOrderConflict(wrapped)
	if !same {
		t.Fatal("expected same=true for wrapped ErrOrderExistsSame")
	}
}
