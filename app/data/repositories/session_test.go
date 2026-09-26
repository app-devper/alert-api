package repositories

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/app-devper/um-api/sessionclient"
)

// Without REDIS_HOST the entity must fail closed, never authorize.
func TestUnconfiguredSessionEntityFailsClosed(t *testing.T) {
	entity := &sessionEntity{}
	userId, err := entity.Authorize(context.Background(), "s1", "SYS", http.MethodGet)
	if !errors.Is(err, sessionclient.ErrUnavailable) || userId != "" {
		t.Fatalf("expected ErrUnavailable and no user, got %q %v", userId, err)
	}
}
