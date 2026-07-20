package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()
		requestID := generateRequestID()
		ctx = context.WithValue(ctx, requestIDKey, requestID)

		logger.InfoContext(ctx, "gRPC request started",
			slog.String("method", info.FullMethod),
			slog.String("request_id", requestID),
		)

		resp, err = handler(ctx, req)

		st := status.Convert(err)
		code := st.Code()

		level := slog.LevelInfo
		if code != codes.OK {
			level = slog.LevelError
		}

		logger.Log(ctx, level, "gRPC request completed",
			slog.String("method", info.FullMethod),
			slog.String("request_id", requestID),
			slog.Duration("duration", time.Since(start)),
			slog.String("code", code.String()),
			slog.String("error", st.Message()),
		)

		return resp, err
	}
}
