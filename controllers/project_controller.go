package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/EsanSamuel/reverse-proxy/db"
	"github.com/EsanSamuel/reverse-proxy/helpers"
	"github.com/EsanSamuel/reverse-proxy/models"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func GenerateApiKey() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		log.Println("Error generating key")
		return ""
	}

	return hex.EncodeToString(b)
}

func CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var project models.Project
	validate := validator.New()

	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		log.Println(err)
		return
	}

	if err := validate.Struct(project); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		log.Println(err)
		return
	}

	project.ProjectID = bson.NewObjectID().Hex()
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()

	result, err := db.Proxy_ProjectCollection.InsertOne(ctx, project)
	if err != nil {
		helpers.ErrorResponse(w, http.StatusInternalServerError, "Error creating project", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"project": result})
}

func ProxyApiKey(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectId := r.URL.Query().Get("projectId")

	api_key := GenerateApiKey()

	updateApiKey := bson.M{
		"$set": bson.M{
			"api_key":    api_key,
			"updated_at": time.Now(),
		},
	}

	result, err := db.Proxy_ProjectCollection.UpdateOne(ctx, bson.M{"project_id": projectId}, updateApiKey)

	if err != nil {
		helpers.ErrorResponse(w, http.StatusInternalServerError, "Error creating project api key", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"project": result})
}

func GetProxyProjects(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userId := r.URL.Query().Get("userId")

	var projects []models.Project

	cursor, err := db.Proxy_ProjectCollection.Find(ctx, bson.M{"user_id": userId})

	if err != nil {
		helpers.ErrorResponse(w, http.StatusInternalServerError, "Error getting user projects", err)
		return
	}

	if err := cursor.All(ctx, &projects); err != nil {
		helpers.ErrorResponse(w, http.StatusInternalServerError, "Error decoding user projects", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"projects": projects})
}

func GetProxyProjectLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectId := r.URL.Query().Get("projectId")

	var logs []models.ResponseLog

	cursor, err := db.Response_Log.Find(ctx, bson.M{"project_id": projectId})

	if err != nil {
		helpers.ErrorResponse(w, http.StatusInternalServerError, "Error getting proxy response log", err)
		return
	}

	if err := cursor.All(ctx, &logs); err != nil {
		helpers.ErrorResponse(w, http.StatusInternalServerError, "Error decoding proxy response log", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"logs": logs})
}
