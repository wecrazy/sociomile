package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthServiceLogin(t *testing.T) {
	fixture := newServiceFixture(t)

	result, err := fixture.authService.Login(fixture.ctx, fixture.adminA.Email, fixture.password)
	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.Equal(t, fixture.adminA.ID, result.User.ID)

	_, err = fixture.authService.Login(fixture.ctx, fixture.adminA.Email, "wrong-password")
	require.Error(t, err)
	requireAppErrorCode(t, err, "invalid_credentials")

	_, err = fixture.authService.Login(fixture.ctx, "missing@example.com", fixture.password)
	require.Error(t, err)
	requireAppErrorCode(t, err, "invalid_credentials")
}
