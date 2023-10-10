package handlers

import (
	"encoding/json"
	"go-upload-file-example/internal/models"
	"go-upload-file-example/internal/services"
	"go-upload-file-example/internal/utils"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	maxFileSize  = 1024 * 2024
	allowedExt   = ".csv"
	uploadFolder = "./public/uploads"
)

func GetFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileIdStr := vars["fileId"]
	fileId, err := uuid.Parse(fileIdStr)
	if err != nil {
		errorResponse := utils.ErrorResponse{Error: "Invalid fileId format"}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}
	file, err := services.GetFileByID(fileId)
	if err != nil {
		errorResponse := utils.ErrorResponse{Error: err.Error()}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
		return
	}

	successResponse := struct {
		Message string       `json:"message"`
		File    *models.File `json:"file"`
	}{
		Message: "success",
		File:    file,
	}
	jsonResponse, _ := json.Marshal(successResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func GetFilesHandler(w http.ResponseWriter, r *http.Request) {
	files := services.GetFiles()

	successResponse := struct {
		Message string                     `json:"message"`
		Files   map[uuid.UUID]*models.File `json:"files"`
	}{
		Message: "success",
		Files:   files,
	}
	jsonResponse, _ := json.Marshal(successResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func CreateFileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	file, _, err := r.FormFile("file")
	if err != nil {
		errorResponse := utils.ErrorResponse{Error: "Failed to get file"}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}
	defer file.Close()

	fileHeader := r.MultipartForm.File["file"][0]
	fileName := fileHeader.Filename

	if fileHeader.Size > maxFileSize {
		errorResponse := utils.ErrorResponse{Error: "File size is too large"}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write(jsonResponse)
		return
	}

	ext := filepath.Ext(fileName)
	if ext != allowedExt {
		errorResponse := utils.ErrorResponse{Error: "Invalid file extension"}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}

	createdFile, err := services.CreateFileService(fileHeader)
	if err != nil {
		errorResponse := utils.ErrorResponse{Error: err.Error()}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}

	successResponse := struct {
		Message string    `json:"message"`
		ID      uuid.UUID `json:"id"`
		Name    string    `json:"name"`
	}{
		Message: "File uploaded successfully",
		ID:      createdFile.ID,
		Name:    createdFile.Name,
	}
	jsonResponse, _ := json.Marshal(successResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonResponse)

}
