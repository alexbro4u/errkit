package errkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// ---- New ----

func TestNew(t *testing.T) {
	err := New("something failed")
	if err.Error() != "something failed" {
		t.Fatalf("unexpected message: %s", err.Error())
	}
}

func TestNewWithOptions(t *testing.T) {
	err := New("not found",
		Code("NOT_FOUND"),
		HTTP(404),
		Retryable(),
		WithSev(SeverityHigh),
		WithFields(String("user_id", "u123")),
	)

	if err.ErrCode() != "NOT_FOUND" {
		t.Fatalf("unexpected code: %s", err.ErrCode())
	}
	if err.Error() != "not found" {
		t.Fatalf("unexpected message: %s", err.Error())
	}
}

// ---- Wrap ----

func TestWrap(t *testing.T) {
	cause := errors.New("db timeout")
	err := Wrap(cause, "failed to fetch user")

	if err.Error() != "failed to fetch user: db timeout" {
		t.Fatalf("unexpected message: %s", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is should match cause")
	}
}

func TestWrapNil(t *testing.T) {
	err := Wrap(nil, "nothing")
	if err != nil {
		t.Fatal("Wrap(nil) should return nil")
	}
}

func TestWrapWithOptions(t *testing.T) {
	cause := errors.New("connection refused")
	err := Wrap(cause, "service unavailable",
		Code("SERVICE_UNAVAILABLE"),
		HTTP(503),
		Retryable(),
	)

	if err.ErrCode() != "SERVICE_UNAVAILABLE" {
		t.Fatalf("unexpected code: %s", err.ErrCode())
	}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is should match cause")
	}
}

// ---- With ----

func TestWith(t *testing.T) {
	err := New("fail", Code("X"))
	err2 := With(err, String("key", "val"))

	if err2 == err {
		t.Fatal("With must return a new error (immutability)")
	}
	v, ok := GetString(err2, "key")
	if !ok || v != "val" {
		t.Fatalf("expected key=val, got %v %v", v, ok)
	}
	// original unchanged
	_, ok = GetString(err, "key")
	if ok {
		t.Fatal("original error should not have the field")
	}
}

func TestWithNil(t *testing.T) {
	err := With(nil, String("k", "v"))
	if err != nil {
		t.Fatal("With(nil) should return nil")
	}
}

// ---- WithCode ----

func TestWithCode(t *testing.T) {
	err := New("fail")
	err2 := WithCode(err, "FAIL_CODE")
	if err2.ErrCode() != "FAIL_CODE" {
		t.Fatalf("unexpected code: %s", err2.ErrCode())
	}
	if err.ErrCode() != "" {
		t.Fatal("original should have no code")
	}
}

// ---- Retryable ----

func TestRetryable(t *testing.T) {
	err := New("timeout", Retryable())
	if !IsRetryable(err) {
		t.Fatal("should be retryable")
	}
}

func TestMarkRetryable(t *testing.T) {
	err := New("timeout")
	if IsRetryable(err) {
		t.Fatal("should not be retryable yet")
	}
	err2 := MarkRetryable(err)
	if !IsRetryable(err2) {
		t.Fatal("should be retryable after mark")
	}
	// immutability
	if IsRetryable(err) {
		t.Fatal("original should not be retryable")
	}
}

func TestNotRetryable(t *testing.T) {
	err := New("fatal", NotRetryable())
	if IsRetryable(err) {
		t.Fatal("should not be retryable")
	}
}

func TestMarkNotRetryable(t *testing.T) {
	err := MarkRetryable(New("timeout"))
	err2 := MarkNotRetryable(err)
	if IsRetryable(err2) {
		t.Fatal("should not be retryable after mark")
	}
}

func TestMarkRetryableNil(t *testing.T) {
	if MarkRetryable(nil) != nil {
		t.Fatal("MarkRetryable(nil) should return nil")
	}
	if MarkNotRetryable(nil) != nil {
		t.Fatal("MarkNotRetryable(nil) should return nil")
	}
}

// ---- Severity ----

func TestSeverity(t *testing.T) {
	err := New("critical issue", WithSev(SeverityCritical))
	s, ok := GetSeverity(err)
	if !ok || s != SeverityCritical {
		t.Fatalf("unexpected severity: %v %v", s, ok)
	}
}

func TestWithSeverity(t *testing.T) {
	err := New("issue")
	err2 := WithSeverity(err, SeverityHigh)
	s, ok := GetSeverity(err2)
	if !ok || s != SeverityHigh {
		t.Fatalf("unexpected severity: %v %v", s, ok)
	}
	_, ok = GetSeverity(err)
	if ok {
		t.Fatal("original should have no severity")
	}
}

func TestWithSeverityNil(t *testing.T) {
	if WithSeverity(nil, SeverityHigh) != nil {
		t.Fatal("WithSeverity(nil) should return nil")
	}
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		s    Severity
		want string
	}{
		{SeverityLow, "low"},
		{SeverityMedium, "medium"},
		{SeverityHigh, "high"},
		{SeverityCritical, "critical"},
		{Severity(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

// ---- HTTP Status ----

func TestHTTPStatus(t *testing.T) {
	err := New("not found", Code("NOT_FOUND"), HTTP(404))
	if status := HTTPStatus(err); status != 404 {
		t.Fatalf("unexpected status: %d", status)
	}
}

func TestHTTPStatusDefault(t *testing.T) {
	err := New("unknown error")
	if status := HTTPStatus(err); status != 500 {
		t.Fatalf("expected 500 default, got: %d", status)
	}
}

func TestWithHTTP(t *testing.T) {
	err := New("err")
	err2 := WithHTTP(err, 400)
	if HTTPStatus(err2) != 400 {
		t.Fatal("expected 400")
	}
	if HTTPStatus(err) != 500 {
		t.Fatal("original should default to 500")
	}
}

func TestWithHTTPNil(t *testing.T) {
	if WithHTTP(nil, 400) != nil {
		t.Fatal("WithHTTP(nil) should return nil")
	}
}

// ---- Stack Trace ----

func TestWithStack(t *testing.T) {
	err := WithStack(New("fail"))
	if err.StackTrace() == nil {
		t.Fatal("expected stack trace")
	}
	if len(err.StackTrace()) == 0 {
		t.Fatal("stack trace should not be empty")
	}
}

func TestStackOption(t *testing.T) {
	err := New("fail", Stack())
	if err.StackTrace() == nil || len(err.StackTrace()) == 0 {
		t.Fatal("expected stack trace from option")
	}
}

func TestWithStackNil(t *testing.T) {
	if WithStack(nil) != nil {
		t.Fatal("WithStack(nil) should return nil")
	}
}

// ---- errors.Is / errors.As ----

func TestErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := Wrap(sentinel, "layer1")
	err2 := Wrap(err, "layer2")

	if !errors.Is(err2, sentinel) {
		t.Fatal("errors.Is should find sentinel through chain")
	}
}

func TestErrorsAs(t *testing.T) {
	err := Wrap(New("inner", Code("INNER")), "outer")
	var e *Error
	if !errors.As(err, &e) {
		t.Fatal("errors.As should find *Error")
	}
}

// ---- CodeIs / GetCode ----

func TestCodeIs(t *testing.T) {
	err := Wrap(New("inner", Code("DB_ERROR")), "outer")
	if !CodeIs(err, "DB_ERROR") {
		t.Fatal("CodeIs should find code through chain")
	}
	if CodeIs(err, "OTHER") {
		t.Fatal("CodeIs should not match unrelated code")
	}
}

func TestGetCode(t *testing.T) {
	err := Wrap(New("inner", Code("DB_ERROR")), "outer", Code("SERVICE_ERROR"))
	if code := GetCode(err); code != "SERVICE_ERROR" {
		t.Fatalf("expected SERVICE_ERROR, got %s", code)
	}
}

func TestGetCodeEmpty(t *testing.T) {
	err := errors.New("plain error")
	if code := GetCode(err); code != "" {
		t.Fatalf("expected empty code, got %s", code)
	}
}

// ---- GetField / GetString / GetInt ----

func TestGetField(t *testing.T) {
	err := New("fail", WithFields(String("key", "val"), Int("count", 5)))
	v, ok := GetField(err, "key")
	if !ok || v != "val" {
		t.Fatalf("unexpected: %v %v", v, ok)
	}
	v, ok = GetField(err, "count")
	if !ok || v != int64(5) {
		t.Fatalf("unexpected: %v %v", v, ok)
	}
}

func TestGetFieldFromChain(t *testing.T) {
	inner := New("inner", WithFields(String("trace_id", "abc")))
	outer := Wrap(inner, "outer")
	v, ok := GetString(outer, "trace_id")
	if !ok || v != "abc" {
		t.Fatalf("expected trace_id=abc from chain, got %v %v", v, ok)
	}
}

func TestGetInt(t *testing.T) {
	err := New("fail", WithFields(Int("attempts", 3)))
	v, ok := GetInt(err, "attempts")
	if !ok || v != 3 {
		t.Fatalf("unexpected: %v %v", v, ok)
	}
}

func TestGetFieldNotFound(t *testing.T) {
	err := New("fail")
	_, ok := GetField(err, "missing")
	if ok {
		t.Fatal("should not find missing field")
	}
}

// ---- Format ----

func TestFormatVerbose(t *testing.T) {
	err := New("fail",
		Code("FAIL"),
		Retryable(),
		WithSev(SeverityHigh),
		HTTP(503),
		WithFields(String("svc", "auth")),
	)
	out := fmt.Sprintf("%+v", err)
	for _, want := range []string{"fail", "code: FAIL", "retryable: true", "severity: high", "http: 503", "svc: auth"} {
		if !strings.Contains(out, want) {
			t.Errorf("verbose format missing %q in:\n%s", want, out)
		}
	}
}

func TestFormatDefault(t *testing.T) {
	err := New("fail")
	if fmt.Sprintf("%v", err) != "fail" {
		t.Fatal("default format should equal Error()")
	}
	if fmt.Sprintf("%s", err) != "fail" {
		t.Fatal("string format should equal Error()")
	}
}

func TestFormatQuoted(t *testing.T) {
	err := New("fail")
	if fmt.Sprintf("%q", err) != `"fail"` {
		t.Fatal("quoted format incorrect")
	}
}

// ---- JSON ----

func TestJSON(t *testing.T) {
	err := New("not found",
		Code("NOT_FOUND"),
		HTTP(404),
		Retryable(),
		WithSev(SeverityHigh),
		WithFields(String("user_id", "u1"), Int("attempt", 2)),
	)

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("marshal error: %v", marshalErr)
	}

	var m map[string]any
	if uerr := json.Unmarshal(data, &m); uerr != nil {
		t.Fatalf("unmarshal error: %v", uerr)
	}

	if m["msg"] != "not found" {
		t.Fatalf("unexpected msg: %v", m["msg"])
	}
	if m["code"] != "NOT_FOUND" {
		t.Fatalf("unexpected code: %v", m["code"])
	}
	if m["retryable"] != true {
		t.Fatalf("unexpected retryable: %v", m["retryable"])
	}
	if m["severity"] != "high" {
		t.Fatalf("unexpected severity: %v", m["severity"])
	}
	if m["http_status"] != float64(404) {
		t.Fatalf("unexpected http_status: %v", m["http_status"])
	}
	fields, ok := m["fields"].(map[string]any)
	if !ok {
		t.Fatalf("fields not a map: %T", m["fields"])
	}
	if fields["user_id"] != "u1" {
		t.Fatalf("unexpected user_id: %v", fields["user_id"])
	}
}

func TestJSONWithCause(t *testing.T) {
	cause := errors.New("db timeout")
	err := Wrap(cause, "service error", Code("TIMEOUT"))
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("marshal error: %v", marshalErr)
	}
	var m map[string]any
	if uerr := json.Unmarshal(data, &m); uerr != nil {
		t.Fatalf("unmarshal error: %v", uerr)
	}
	if m["cause"] != "db timeout" {
		t.Fatalf("unexpected cause: %v", m["cause"])
	}
}

// ---- slog.LogValuer ----

func TestLogValue(t *testing.T) {
	err := New("not found",
		Code("NOT_FOUND"),
		WithFields(String("user_id", "u1")),
	)

	val := err.LogValue()
	if val.Kind() != slog.KindGroup {
		t.Fatalf("expected group, got %v", val.Kind())
	}

	attrs := val.Group()
	found := make(map[string]bool)
	for _, a := range attrs {
		found[a.Key] = true
	}
	for _, key := range []string{"msg", "code", "user_id"} {
		if !found[key] {
			t.Errorf("missing key %q in LogValue", key)
		}
	}
}

// ---- Immutability ----

func TestImmutability(t *testing.T) {
	err := New("base", Code("BASE"), WithFields(String("k", "v")))
	err2 := With(err, String("k2", "v2"))
	err3 := WithCode(err, "CHANGED")

	// original unchanged
	if err.ErrCode() != "BASE" {
		t.Fatal("original code should not change")
	}
	if len(err.Fields()) != 1 {
		t.Fatalf("original should have 1 field, got %d", len(err.Fields()))
	}
	// copies have changes
	if len(err2.Fields()) != 2 {
		t.Fatalf("err2 should have 2 fields, got %d", len(err2.Fields()))
	}
	if err3.ErrCode() != "CHANGED" {
		t.Fatal("err3 code should be CHANGED")
	}
}

// ---- With on plain error ----

func TestWithPlainError(t *testing.T) {
	plain := errors.New("plain")
	err := With(plain, String("key", "val"))
	if err == nil {
		t.Fatal("should not be nil")
	}
	v, ok := GetString(err, "key")
	if !ok || v != "val" {
		t.Fatalf("expected key=val, got %v %v", v, ok)
	}
	// should still chain to original
	if !errors.Is(err, plain) {
		t.Fatal("should chain to original plain error")
	}
}

// ---- WithCode on nil ----

func TestWithCodeNil(t *testing.T) {
	if WithCode(nil, "X") != nil {
		t.Fatal("WithCode(nil) should return nil")
	}
}

// ---- Field types ----

func TestFieldTypes(t *testing.T) {
	err := New("test",
		WithFields(
			String("s", "hello"),
			Int("i", 42),
			Int64("i64", 999),
			Bool("b", true),
			Float64("f", 3.14),
			Any("a", []int{1, 2}),
		),
	)

	fields := err.Fields()
	if len(fields) != 6 {
		t.Fatalf("expected 6 fields, got %d", len(fields))
	}

	if v := fields[0].Value(); v != "hello" {
		t.Errorf("string field: %v", v)
	}
	if v := fields[1].Value(); v != int64(42) {
		t.Errorf("int field: %v", v)
	}
	if v := fields[2].Value(); v != int64(999) {
		t.Errorf("int64 field: %v", v)
	}
	if v := fields[3].Value(); v != true {
		t.Errorf("bool field: %v", v)
	}
	if v := fields[4].Value(); v != 3.14 {
		t.Errorf("float64 field: %v", v)
	}
	if v, ok := fields[5].Value().([]int); !ok || len(v) != 2 {
		t.Errorf("any field: %v", fields[5].Value())
	}
}

// ---- Message accessor ----

func TestMessage(t *testing.T) {
	cause := errors.New("root")
	err := Wrap(cause, "wrapper")
	if err.Message() != "wrapper" {
		t.Fatalf("Message() should return own msg, got: %s", err.Message())
	}
	if err.Error() != "wrapper: root" {
		t.Fatalf("Error() should include cause, got: %s", err.Error())
	}
}

// ---- Deep chain ----

func TestDeepChain(t *testing.T) {
	e0 := New("level0", Code("L0"), WithFields(String("depth", "0")))
	e1 := Wrap(e0, "level1", Code("L1"))
	e2 := Wrap(e1, "level2")
	e3 := Wrap(e2, "level3", Retryable(), HTTP(502))

	if !CodeIs(e3, "L0") {
		t.Fatal("should find L0 in chain")
	}
	if !CodeIs(e3, "L1") {
		t.Fatal("should find L1 in chain")
	}
	if !IsRetryable(e3) {
		t.Fatal("should be retryable")
	}
	if HTTPStatus(e3) != 502 {
		t.Fatal("should be 502")
	}
	v, ok := GetString(e3, "depth")
	if !ok || v != "0" {
		t.Fatal("should find depth=0 in deep chain")
	}
}
