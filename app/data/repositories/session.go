package repositories

import (
	"context"
	"encoding/json"
	"time"

	"alert/db"
)

const sessionPrefix = "session:"

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
	raw, err := entity.resource.RdDb.Get(ctx, sessionPrefix+sessionId).Result()
	if err != nil {
		return "", err
	}
	var data struct {
		UserId string `json:"userId"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return "", err
	}
	return data.UserId, nil
}
