package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kirian-dev/go-import-file-csv/internal/handlers"
	"github.com/kirian-dev/go-import-file-csv/scripts"

	"github.com/gorilla/mux"
)

const (
	PORT = ":8080"
)

func main() {
	Run()
}

func Run() {
	r := mux.NewRouter()

	r.HandleFunc("/api/files", handlers.GetFilesHandler).Methods("GET")
	r.HandleFunc("/api/files/{fileId}", handlers.GetFileHandler).Methods("GET")
	r.HandleFunc("/api/files", handlers.CreateFileHandler).Methods("POST")

	http.Handle("/", r)

	//generate test cvs file
	fileName := "test_accounts.csv"
	numAccounts := 1000
	folderPath := "public"

	err := scripts.GenerateCSV(fileName, numAccounts, folderPath)
	if err != nil {
		log.Println("Error generating test cvs file")
		panic(err)
	}

	fmt.Println("Server running on port:", PORT)
	http.ListenAndServe(PORT, nil)
}
