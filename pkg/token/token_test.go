package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPasetoMaker_CreateAndVerifyToken(t *testing.T) {
	maker, err := NewPasetoMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	userID := int64(123)
	duration := time.Hour

	token, payload, err := maker.CreateToken(userID, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, userID, payload.UserID)
	require.WithinDuration(t, time.Now(), payload.IssuedAt, time.Second*2)
	require.WithinDuration(t, time.Now().Add(duration), payload.ExpiredAt, time.Second*2)

	// Verify token
	verifiedPayload, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.Equal(t, payload.UserID, verifiedPayload.UserID)
	require.WithinDuration(t, payload.IssuedAt, verifiedPayload.IssuedAt, time.Second)
	require.WithinDuration(t, payload.ExpiredAt, verifiedPayload.ExpiredAt, time.Second)
}

func TestPasetoMaker_VerifyExpiredToken(t *testing.T) {
	maker, err := NewPasetoMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	// Create token that expires immediately
	token, _, err := maker.CreateToken(123, -time.Hour)
	require.NoError(t, err)

	_, err = maker.VerifyToken(token)
	require.Error(t, err)
	require.Equal(t, "token has expired", err.Error())
}

func TestPasetoMaker_VerifyInvalidToken(t *testing.T) {
	maker, err := NewPasetoMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	_, err = maker.VerifyToken("invalid-token")
	require.Error(t, err)
}

func TestPasetoMaker_WrongKey(t *testing.T) {
	maker1, err := NewPasetoMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	maker2, err := NewPasetoMaker("22345678901234567890123456789012")
	require.NoError(t, err)

	token, _, err := maker1.CreateToken(123, time.Hour)
	require.NoError(t, err)

	_, err = maker2.VerifyToken(token)
	require.Error(t, err)
}

func TestNewPasetoMaker_InvalidKeyLength(t *testing.T) {
	_, err := NewPasetoMaker("short-key")
	require.Error(t, err)
	require.Equal(t, "symmetric key must be 32 bytes", err.Error())
}