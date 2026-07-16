package repositories

import (
	"context"
	"time"

	"alert/db"
)

type sessionEntity struct {
	resource *db.Resource
}

type ISession interface {
	GetSessionById(sessionId string) (string, error)
}

func NewSessionEntity(resource *db.Resource) ISession {
	return &sessionEntity{resource: resource}
}

func (entity *sessionEntity) GetSessionById(sessionId string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return entity.resource.RdDb.Get(ctx, sessionId).Result()
}
