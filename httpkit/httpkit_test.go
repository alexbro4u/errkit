package httpkit

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexbro4u/errkit"
)

func TestWriteError(t *testing.T) {
	err := errkit.New("user not found",
		errkit.Code("NOT_FOUND"),
		errkit.HTTP(404),
		errkit.WithFields(errkit.String("user_id", "u1")),
	)

	w := httptest.NewRecorder()
	WriteError(w, err)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", ct)
	}

	var body ErrorBody
	if uerr := json.NewDecoder(w.Body).Decode(&body); uerr != nil {
		t.Fatalf("decode error: %v", uerr)
	}
	if body.Error != "user not found" {
		t.Fatalf("unexpected error: %s", body.Error)
	}
	if body.Code != "NOT_FOUND" {
		t.Fatalf("unexpected code: %s", body.Code)
	}
	if body.Fields["user_id"] != "u1" {
		t.Fatalf("unexpected fields: %v", body.Fields)
	}
}

func TestWriteErrorDefault500(t *testing.T) {
	err := errkit.New("unknown error")

	w := httptest.NewRecorder()
	WriteError(w, err)

	if w.Code != 500 {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestWriteErrorWithStatus(t *testing.T) {
	err := errkit.New("bad request", errkit.HTTP(400))

	w := httptest.NewRecorder()
	WriteErrorWithStatus(w, 422, err)

	if w.Code != 422 {
		t.Fatalf("expected 422 override, got %d", w.Code)
	}
}

func TestWriteErrorPlainError(t *testing.T) {
	err := errors.New("plain error")

	w := httptest.NewRecorder()
	WriteError(w, err)

	if w.Code != 500 {
		t.Fatalf("expected 500 for plain error, got %d", w.Code)
	}

	var body ErrorBody
	if uerr := json.NewDecoder(w.Body).Decode(&body); uerr != nil {
		t.Fatalf("decode error: %v", uerr)
	}
	if body.Error != "plain error" {
		t.Fatalf("unexpected error: %s", body.Error)
	}
}

func TestWriteErrorWrapped(t *testing.T) {
	inner := errkit.New("db timeout", errkit.Code("DB_TIMEOUT"), errkit.HTTP(503))
	outer := errkit.Wrap(inner, "service failed")

	w := httptest.NewRecorder()
	WriteError(w, outer)

	if w.Code != 503 {
		t.Fatalf("expected 503 from chain, got %d", w.Code)
	}

	var body ErrorBody
	if uerr := json.NewDecoder(w.Body).Decode(&body); uerr != nil {
		t.Fatalf("decode error: %v", uerr)
	}
	if body.Code != "DB_TIMEOUT" {
		t.Fatalf("unexpected code: %s", body.Code)
	}
}

func TestHandler(t *testing.T) {
	h := Handler(func(w http.ResponseWriter, r *http.Request) error {
		return errkit.New("forbidden",
			errkit.Code("FORBIDDEN"),
			errkit.HTTP(403),
		)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	h.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	var body ErrorBody
	if uerr := json.NewDecoder(w.Body).Decode(&body); uerr != nil {
		t.Fatalf("decode error: %v", uerr)
	}
	if body.Code != "FORBIDDEN" {
		t.Fatalf("unexpected code: %s", body.Code)
	}
}

func TestHandlerNoError(t *testing.T) {
	h := Handler(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestMiddlewarePanicRecovery(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	h := Middleware(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/panic", nil)
	h.ServeHTTP(w, r)

	if w.Code != 500 {
		t.Fatalf("expected 500 after panic, got %d", w.Code)
	}

	var body ErrorBody
	if uerr := json.NewDecoder(w.Body).Decode(&body); uerr != nil {
		t.Fatalf("decode error: %v", uerr)
	}
	if body.Code != "INTERNAL" {
		t.Fatalf("unexpected code: %s", body.Code)
	}
}

func TestMiddlewareNoPanic(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	h := Middleware(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/ok", nil)
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
