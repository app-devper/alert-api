package repositories

import (
	"context"
	"errors"

	"alert/db"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/sirupsen/logrus"
)

// ISession confirms the UM session behind a verified access token by reading
// UM's session store through sessionclient (um-api ADR-0004).
type ISession interface {
	// Authorize returns the session's user id. It fails with
	// sessionclient.ErrSessionRejected when the session is gone or belongs to
	// another system, and with sessionclient.ErrUnavailable when the store
	// cannot answer and the outage policy does not let the request continue.
	Authorize(ctx context.Context, sessionId, system, method string) (string, error)
}

type sessionEntity struct {
	checker *sessionclient.Checker
}

// NewSessionEntity reads UM sessions from REDIS_HOST. If that is missing or
// invalid it fails closed: every check reports the store as unavailable.
func NewSessionEntity(resource *db.Resource) ISession {
	checker, err := sessionclient.New(resource.RedisHost)
	if err != nil || !checker.Enabled() {
		logrus.Errorf("session client not configured (REDIS_HOST): %v; every session check will fail", err)
		return &sessionEntity{}
	}
	return &sessionEntity{checker: checker}
}

func (entity *sessionEntity) Authorize(ctx context.Context, sessionId, system, method string) (string, error) {
	if entity.checker == nil {
		return "", sessionclient.ErrUnavailable
	}
	session, err := entity.checker.Authorize(ctx, sessionId, system, method)
	if err != nil {
		if !errors.Is(err, sessionclient.ErrSessionRejected) {
			logrus.Warn("session check: ", err)
		}
		return "", err
	}
	return session.UserId, nil
}
