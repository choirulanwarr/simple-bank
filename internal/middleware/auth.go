package middleware

import (
	"context"
	"strings"

	"github.com/choirulanwar/simple-bank/pkg/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthInterceptor(tokenMaker token.Maker) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// Skip auth for public endpoints
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		values := md["authorization"]
		if len(values) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization token")
		}

		authHeader := values[0]
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization format")
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == "" {
			return nil, status.Errorf(codes.Unauthenticated, "empty token")
		}

		payload, err := tokenMaker.VerifyToken(tokenStr)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid or expired token: %v", err)
		}

		// Inject user info into context
		ctx = context.WithValue(ctx, userIDKey, payload.UserID)

		return handler(ctx, req)
	}
}

type contextKey string

const userIDKey contextKey = "user_id"

func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

func isPublicMethod(fullMethod string) bool {
	publicMethods := map[string]bool{
		"/simplebank.SimpleBank/Login":          true,
		"/simplebank.SimpleBank/CreateCustomer": true,
	}

	return publicMethods[fullMethod]
}
