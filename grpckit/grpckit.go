// Package grpckit provides gRPC helpers for errkit errors.
//
// It converts errkit errors into gRPC status errors with the appropriate
// status code and structured details.
package grpckit

import (
	"context"
	"errors"
	"net/http"

	"github.com/alexbro4u/errkit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// httpStatusClientClosedRequest is the non-standard HTTP status code for client closed request (nginx).
const httpStatusClientClosedRequest = 499

// Error converts an errkit error into a gRPC status error.
// The gRPC code is derived from the HTTP status on the error (see HTTPToGRPC).
// The message is the error's Error() string.
// If err is nil, returns nil.
func Error(err error) error {
	if err == nil {
		return nil
	}
	code := HTTPToGRPC(errkit.HTTPStatus(err))
	return status.Error(code, err.Error())
}

// Status converts an errkit error into a *status.Status.
// If err is nil, returns status.New(codes.OK, "").
func Status(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}
	code := HTTPToGRPC(errkit.HTTPStatus(err))
	return status.New(code, err.Error())
}

// FromStatus converts a gRPC status error back into an *errkit.Error.
// If err is not a gRPC status error, it wraps it as-is.
// If err is nil, returns nil.
func FromStatus(err error) *errkit.Error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return errkit.Wrap(err, err.Error())
	}
	httpCode := GRPCToHTTP(st.Code())
	return errkit.New(st.Message(),
		errkit.Code(st.Code().String()),
		errkit.HTTP(httpCode),
	)
}

// IsGRPCError checks if the error is a gRPC status error with the given code.
func IsGRPCError(err error, code codes.Code) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	return st.Code() == code
}

// UnaryServerInterceptor returns a gRPC unary server interceptor that
// converts errkit errors returned by handlers into proper gRPC status errors.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			var ek *errkit.Error
			if errors.As(err, &ek) {
				return resp, Error(err)
			}
		}
		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor that
// converts errkit errors returned by handlers into proper gRPC status errors.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, ss)
		if err != nil {
			var ek *errkit.Error
			if errors.As(err, &ek) {
				return Error(err)
			}
		}
		return err
	}
}

// HTTPToGRPC maps an HTTP status code to the closest gRPC status code.
func HTTPToGRPC(httpStatus int) codes.Code {
	switch httpStatus {
	case http.StatusOK:
		return codes.OK
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusRequestTimeout:
		return codes.DeadlineExceeded
	case http.StatusGone:
		return codes.NotFound
	case http.StatusPreconditionFailed:
		return codes.FailedPrecondition
	case http.StatusUnprocessableEntity:
		return codes.InvalidArgument
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	case http.StatusPaymentRequired:
		return codes.FailedPrecondition
	default:
		if httpStatus >= 200 && httpStatus < 300 {
			return codes.OK
		}
		if httpStatus >= 400 && httpStatus < 500 {
			return codes.InvalidArgument
		}
		return codes.Internal
	}
}

// GRPCToHTTP maps a gRPC status code to the closest HTTP status code.
func GRPCToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return httpStatusClientClosedRequest
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
