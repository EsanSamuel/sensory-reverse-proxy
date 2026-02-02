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

type ResponseLog struct {
	ID            bson.ObjectID `json:"_id" bson:"_id,omitempty"`
	ProjectID     string        `json:"project_id" bson:"project_id"`
	ResponseLogId string        `json:"response_log_id" bson:"response_log_id"`
	UserID        string        `json:"user_id" bson:"user_id"`
	Host          string        `json:"host" bson:"host"`
	Method        string        `json:"method" bson:"method"`
	UrlPath       string        `json:"url_path" bson:"url_path"`
	StatusCode    int           `json:"status_code" bson:"status_code"`
	BytesWritten  int64         `json:"bytes_written" bson:"bytes_written" `
	Duration      int64         `json:"duration" bson:"duration"`
	ClientIP      string        `json:"client_ip" bson:"client_ip"`
	UserAgent     string        `json:"user_agent" bson:"user_agent"`
	QueryParams   string        `json:"query_params" bson:"query_params"`
	Referer       string        `json:"referer" bson:"referer"`
	Timestamp     time.Time     `json:"timestamp" bson:"timestamp"`
	Protocol      string        `json:"protocol" bson:"protocol"`
	ContentType   string        `json:"content_type" bson:"content_type"`
}
