package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/choirulanwar/simple-bank/pkg/token"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor(t *testing.T) {
	maker, err := token.NewPasetoMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	interceptor := AuthInterceptor(maker)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{FullMethod: "/simplebank.SimpleBank/GetCustomer"}

	// Test 1: No metadata
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	_, err = interceptor(ctx, req, info, handler)
	require.Error(t, err)
	st := status.Convert(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Contains(t, st.Message(), "missing metadata")

	// Test 2: No authorization header
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("other", "value"))
	_, err = interceptor(ctx, req, info, handler)
	require.Error(t, err)
	st = status.Convert(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Contains(t, st.Message(), "missing authorization token")

	// Test 3: Invalid authorization format
	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "InvalidFormat"))
	_, err = interceptor(ctx, req, info, handler)
	require.Error(t, err)
	st = status.Convert(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Contains(t, st.Message(), "invalid authorization format")

	// Test 4: Empty token
	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "))
	_, err = interceptor(ctx, req, info, handler)
	require.Error(t, err)
	st = status.Convert(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Contains(t, st.Message(), "empty token")

	// Test 5: Invalid token
	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer invalid-token"))
	_, err = interceptor(ctx, req, info, handler)
	require.Error(t, err)
	st = status.Convert(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Contains(t, st.Message(), "invalid or expired token")

	// Test 6: Valid token
	token, payload, err := maker.CreateToken(123, time.Hour)
	require.NoError(t, err)

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	var capturedUserID int64

	handlerWithCapture := func(ctx context.Context, req any) (any, error) {
		userID, ok := GetUserIDFromContext(ctx)
		require.True(t, ok)
		capturedUserID = userID
		return "ok", nil
	}

	_, err = interceptor(ctx, req, info, handlerWithCapture)
	require.NoError(t, err)
	require.Equal(t, payload.UserID, capturedUserID)

	// Test 7: Expired token
	expiredToken, _, err := maker.CreateToken(456, -time.Hour)
	require.NoError(t, err)

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+expiredToken))
	_, err = interceptor(ctx, req, info, handler)
	require.Error(t, err)
	st = status.Convert(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Contains(t, st.Message(), "expired")

	// Test 8: Public method bypasses auth
	publicInfo := &grpc.UnaryServerInfo{FullMethod: "/simplebank.SimpleBank/Login"}
	ctx = context.Background() // no metadata
	_, err = interceptor(ctx, req, publicInfo, handler)
	require.NoError(t, err)

	publicInfo2 := &grpc.UnaryServerInfo{FullMethod: "/simplebank.SimpleBank/CreateCustomer"}
	_, err = interceptor(ctx, req, publicInfo2, handler)
	require.NoError(t, err)
}

func TestGetUserIDFromContext(t *testing.T) {
	ctx := context.Background()
	userID, ok := GetUserIDFromContext(ctx)
	require.False(t, ok)
	require.Equal(t, int64(0), userID)

	ctx = context.WithValue(context.Background(), userIDKey, int64(123))
	userID, ok = GetUserIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(123), userID)
}