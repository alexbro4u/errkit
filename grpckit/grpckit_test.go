package grpckit

import (
	"net/http"
	"testing"

	"github.com/alexbro4u/errkit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestError(t *testing.T) {
	err := errkit.New("not found",
		errkit.Code("NOT_FOUND"),
		errkit.HTTP(404),
	)

	grpcErr := Error(err)
	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", st.Code())
	}
	if st.Message() != "not found" {
		t.Fatalf("unexpected message: %s", st.Message())
	}
}

func TestErrorNil(t *testing.T) {
	if Error(nil) != nil {
		t.Fatal("Error(nil) should return nil")
	}
}

func TestErrorDefault500(t *testing.T) {
	err := errkit.New("unknown")
	grpcErr := Error(err)
	st, _ := status.FromError(grpcErr)
	if st.Code() != codes.Internal {
		t.Fatalf("expected Internal for default 500, got %v", st.Code())
	}
}

func TestStatus(t *testing.T) {
	err := errkit.New("bad request", errkit.HTTP(400))
	st := Status(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestStatusNil(t *testing.T) {
	st := Status(nil)
	if st.Code() != codes.OK {
		t.Fatalf("expected OK for nil, got %v", st.Code())
	}
}

func TestFromStatus(t *testing.T) {
	grpcErr := status.Error(codes.NotFound, "user not found")
	ek := FromStatus(grpcErr)
	if ek == nil {
		t.Fatal("expected non-nil error")
	}
	if ek.ErrCode() != codes.NotFound.String() {
		t.Fatalf("unexpected code: %s", ek.ErrCode())
	}
	if errkit.HTTPStatus(ek) != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", errkit.HTTPStatus(ek))
	}
}

func TestFromStatusNil(t *testing.T) {
	if FromStatus(nil) != nil {
		t.Fatal("FromStatus(nil) should return nil")
	}
}

func TestIsGRPCError(t *testing.T) {
	grpcErr := status.Error(codes.PermissionDenied, "denied")
	if !IsGRPCError(grpcErr, codes.PermissionDenied) {
		t.Fatal("expected PermissionDenied")
	}
	if IsGRPCError(grpcErr, codes.NotFound) {
		t.Fatal("should not match NotFound")
	}
}

func TestIsGRPCErrorNonGRPC(t *testing.T) {
	err := errkit.New("plain")
	if IsGRPCError(err, codes.Internal) {
		t.Fatal("non-gRPC error should return false")
	}
}

func TestHTTPToGRPC(t *testing.T) {
	tests := []struct {
		http int
		grpc codes.Code
	}{
		{200, codes.OK},
		{400, codes.InvalidArgument},
		{401, codes.Unauthenticated},
		{403, codes.PermissionDenied},
		{404, codes.NotFound},
		{409, codes.AlreadyExists},
		{412, codes.FailedPrecondition},
		{422, codes.InvalidArgument},
		{429, codes.ResourceExhausted},
		{408, codes.DeadlineExceeded},
		{410, codes.NotFound},
		{500, codes.Internal},
		{501, codes.Unimplemented},
		{502, codes.Internal},
		{503, codes.Unavailable},
		{504, codes.DeadlineExceeded},
		{402, codes.FailedPrecondition},
		{201, codes.OK},
		{418, codes.InvalidArgument},
	}
	for _, tt := range tests {
		got := HTTPToGRPC(tt.http)
		if got != tt.grpc {
			t.Errorf("HTTPToGRPC(%d) = %v, want %v", tt.http, got, tt.grpc)
		}
	}
}

func TestGRPCToHTTP(t *testing.T) {
	tests := []struct {
		grpc codes.Code
		http int
	}{
		{codes.OK, 200},
		{codes.Canceled, 499},
		{codes.Unknown, 500},
		{codes.InvalidArgument, 400},
		{codes.DeadlineExceeded, 504},
		{codes.NotFound, 404},
		{codes.AlreadyExists, 409},
		{codes.PermissionDenied, 403},
		{codes.ResourceExhausted, 429},
		{codes.FailedPrecondition, 412},
		{codes.Aborted, 409},
		{codes.OutOfRange, 400},
		{codes.Unimplemented, 501},
		{codes.Internal, 500},
		{codes.Unavailable, 503},
		{codes.DataLoss, 500},
		{codes.Unauthenticated, 401},
	}
	for _, tt := range tests {
		got := GRPCToHTTP(tt.grpc)
		if got != tt.http {
			t.Errorf("GRPCToHTTP(%v) = %d, want %d", tt.grpc, got, tt.http)
		}
	}
}

func TestErrorHTTPMappings(t *testing.T) {
	tests := []struct {
		name     string
		http     int
		wantCode codes.Code
	}{
		{"bad request", 400, codes.InvalidArgument},
		{"unauthorized", 401, codes.Unauthenticated},
		{"forbidden", 403, codes.PermissionDenied},
		{"not found", 404, codes.NotFound},
		{"conflict", 409, codes.AlreadyExists},
		{"unavailable", 503, codes.Unavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errkit.New(tt.name, errkit.HTTP(tt.http))
			grpcErr := Error(err)
			st, _ := status.FromError(grpcErr)
			if st.Code() != tt.wantCode {
				t.Errorf("Error(HTTP=%d): got %v, want %v", tt.http, st.Code(), tt.wantCode)
			}
		})
	}
}

func TestFromStatusRoundTrip(t *testing.T) {
	orig := errkit.New("not found", errkit.Code("NOT_FOUND"), errkit.HTTP(404))
	grpcErr := Error(orig)
	recovered := FromStatus(grpcErr)

	if errkit.HTTPStatus(recovered) != 404 {
		t.Fatalf("expected 404 after round-trip, got %d", errkit.HTTPStatus(recovered))
	}
}
