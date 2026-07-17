package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()

		method := info.FullMethod
		logger.InfoContext(ctx, "gRPC request started",
			slog.String("method", method),
			slog.String("type", "unary"),
		)

		resp, err = handler(ctx, req)

		duration := time.Since(start)
		st := status.Convert(err)
		code := st.Code()

		level := slog.LevelInfo
		if code != codes.OK {
			level = slog.LevelError
		}

		logger.Log(ctx, level, "gRPC request completed",
			slog.String("method", method),
			slog.Duration("duration", duration),
			slog.String("code", code.String()),
			slog.String("error", st.Message()),
		)

		return resp, err
	}
}