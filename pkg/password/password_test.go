package password

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	password := "SecureP@ss1"
	hashed, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashed)
	require.NotEqual(t, password, hashed)

	// Verify the hash
	err = VerifyPassword(password, hashed)
	require.NoError(t, err)
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	password := "SecureP@ss1"
	hashed, _ := HashPassword(password)
	err := VerifyPassword("wrongpassword", hashed)
	require.Error(t, err)
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	err := VerifyPassword("password", "invalid-hash")
	require.Error(t, err)
}

func TestHashPassword_SamePasswordDifferentHashes(t *testing.T) {
	password := "SecureP@ss1"
	hashed1, _ := HashPassword(password)
	hashed2, _ := HashPassword(password)
	require.NotEqual(t, hashed1, hashed2) // bcrypt uses salt, so hashes should differ
}
