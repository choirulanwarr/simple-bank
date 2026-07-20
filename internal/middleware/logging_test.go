package middleware

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
)

func TestLoggingInterceptor(t *testing.T) {
	var loggedMethod string
	var loggedDuration time.Duration
	var loggedCode codes.Code

	mockLogger := slog.New(slog.NewJSONHandler(&testWriter{writeFunc: func(p []byte) (n int, err error) {
		var logEntry map[string]any
		_ = json.Unmarshal(p, &logEntry)

		switch msg := logEntry["msg"].(string); msg {
		case "gRPC request started":
			if m, ok := logEntry["method"].(string); ok {
				loggedMethod = m
			}
		case "gRPC request completed":
			if d, ok := logEntry["duration"].(float64); ok {
				loggedDuration = time.Duration(d)
			}
			if c, ok := logEntry["code"].(string); ok {
				loggedCode = parseCode(c)
			}
		}
		return len(p), nil
	}}, &slog.HandlerOptions{}))

	interceptor := LoggingInterceptor(mockLogger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}
	handler := func(ctx context.Context, req any) (any, error) {
		return "test-response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	require.NoError(t, err)
	assert.Equal(t, "test-response", resp)
	assert.Equal(t, "/test.Service/TestMethod", loggedMethod)
	assert.True(t, loggedDuration > 0)
	assert.Equal(t, codes.OK, loggedCode)
}

func TestLoggingInterceptor_WithError(t *testing.T) {
	var loggedCode codes.Code

	mockLogger := slog.New(slog.NewJSONHandler(&testWriter{writeFunc: func(p []byte) (n int, err error) {
		var logEntry map[string]any
		_ = json.Unmarshal(p, &logEntry)

		if msg, ok := logEntry["msg"].(string); ok && msg == "gRPC request completed" {
			if c, ok := logEntry["code"].(string); ok {
				loggedCode = parseCode(c)
			}
		}
		return len(p), nil
	}}, &slog.HandlerOptions{}))

	interceptor := LoggingInterceptor(mockLogger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}

	resp, err := interceptor(ctx, req, info, handler)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, loggedCode)
}

func parseCode(s string) codes.Code {
	switch s {
	case "OK":
		return codes.OK
	case "NotFound":
		return codes.NotFound
	case "Internal":
		return codes.Internal
	case "InvalidArgument":
		return codes.InvalidArgument
	case "FailedPrecondition":
		return codes.FailedPrecondition
	case "AlreadyExists":
		return codes.AlreadyExists
	case "Unauthenticated":
		return codes.Unauthenticated
	case "PermissionDenied":
		return codes.PermissionDenied
	default:
		return codes.Unknown
	}
}

func TestRecoveryInterceptor(t *testing.T) {
	interceptor := RecoveryInterceptor()

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}

	// Test normal execution
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	// Test panic recovery
	handlerPanic := func(ctx context.Context, req any) (any, error) {
		panic("test panic")
	}

	resp, err = interceptor(ctx, req, info, handlerPanic)
	require.Error(t, err)
	assert.Nil(t, resp)
	st := status.Convert(err)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "panic recovered")
	assert.Contains(t, st.Message(), "test panic")
}

type testWriter struct {
	writeFunc func(p []byte) (n int, err error)
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return w.writeFunc(p)
}
