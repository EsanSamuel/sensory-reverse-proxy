package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Project struct {
	ID          bson.ObjectID `json:"_id" bson:"_id,omitempty"`
	ProjectID   string        `json:"project_id" bson:"project_id"`
	ProjectName string        `json:"project_name" bson:"project_name" validate:"required,min=2,max=100"`
	Description string        `json:"description" bson:"description"`
	UserID      string        `json:"user_id" bson:"user_id"`
	ApiKey      string        `json:"api_key" bson:"api_key"`
	CreatedAt   time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" bson:"updated_at"`
	BackendUrls []string      `json:"backend_urls" bson:"backend_urls"`
}
