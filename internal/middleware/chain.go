package middleware

import (
	"context"

	"google.golang.org/grpc"
)

// ChainUnaryInterceptors chains multiple unary interceptors into one
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Build the chain from the end to the beginning
		chained := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			current := interceptors[i]
			next := chained
			chained = func(ctx context.Context, req any) (any, error) {
				return current(ctx, req, info, next)
			}
		}
		return chained(ctx, req)
	}
}
