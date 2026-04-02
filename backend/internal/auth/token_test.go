package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/model"
)

func TestGenerateAndParseTokenBranches(t *testing.T) {
	user := &model.User{
		BaseModel: model.BaseModel{ID: "user-1"},
		TenantID:  "tenant-1",
		Email:     "user@example.com",
		Role:      model.RoleAdmin,
	}

	tokenString, err := GenerateToken("secret", time.Minute, user)
	require.NoError(t, err)

	claims, err := ParseToken("secret", tokenString)
	require.NoError(t, err)
	require.Equal(t, user.ID, claims.UserID)
	require.Equal(t, user.TenantID, claims.TenantID)
	require.Equal(t, user.Email, claims.Email)
	require.Equal(t, user.Role, claims.Role)

	_, err = ParseToken("other-secret", tokenString)
	require.Error(t, err)

	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: user.ID})
	noneString, err := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = ParseToken("secret", noneString)
	require.ErrorContains(t, err, "unexpected signing method")
}
