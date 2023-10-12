package services

import (
	"encoding/csv"
	"fmt"
	"go-upload-file-example/internal/models"
	"go-upload-file-example/internal/utils"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ProcessTask struct {
	Line   []string
	FileID uuid.UUID
}

type Worker struct {
	ID        int
	TaskQueue chan ProcessTask
	Done      chan bool
	WaitGroup *sync.WaitGroup
}

const (
	uploadFolder = "./public/uploads"
	numWorkers   = 4
	bufferSize   = 10
)

var (
	accountsMap         = make(map[string]*models.Account)
	filesMap            = make(map[uuid.UUID]*models.File)
	loadingAccountsZero = make(chan struct{})
	accountsMapMutex    sync.Mutex
	filesMapMutex       sync.Mutex
)

func GetFileByID(id uuid.UUID) (*models.File, error) {
	f, exists := filesMap[id]
	if !exists {
		log.Println("couldn't get file")
		return nil, fmt.Errorf("error getting file")
	}
	return f, nil
}

func GetFiles() map[uuid.UUID]*models.File {
	return filesMap
}

func CreateFileService(file *multipart.FileHeader) (*models.FileResponse, error) {
	tempFileName := utils.GenerateUniqueFileName(file.Filename)
	tempFilePath := filepath.Join(uploadFolder, tempFileName)

	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	dest, err := os.Create(tempFilePath)
	if err != nil {
		return nil, err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		return nil, err
	}

	rows, err := utils.CountLinesInFile(tempFilePath)
	if err != nil {
		return nil, err
	}

	id, err := processFile(tempFilePath, tempFileName, rows)
	if err != nil {
		return nil, err
	}

	return &models.FileResponse{
		Name: tempFileName,
		ID:   id,
	}, nil
}

func updateFileStatus(id uuid.UUID) {
	filesMapMutex.Lock()
	defer filesMapMutex.Unlock()
	f, exists := filesMap[id]
	if exists && f.LoadingAccounts == 0 {
		f.Status = models.SuccessStatus
		f.EndTime = time.Now()
	}
}

func processFile(filePath, tempFileName string, rows int) (uuid.UUID, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return uuid.Nil, err
	}
	defer file.Close()

	newFile := &models.File{
		ID:              uuid.New(),
		Name:            tempFileName,
		SuccessAccounts: 0,
		FailAccounts:    0,
		LoadingAccounts: rows,
		CreatedAt:       time.Now(),
		EndTime:         time.Time{},
		Status:          models.LoadingStatus,
	}

	filesMapMutex.Lock()
	filesMap[newFile.ID] = newFile
	filesMapMutex.Unlock()

	waitGroup := &sync.WaitGroup{}
	taskChannel := make(chan ProcessTask, bufferSize)

	workers := CreateJobPool(numWorkers, taskChannel, waitGroup)

	reader := csv.NewReader(file)
	skipHeader := true
	for {
		line, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error reading line: %v", err)
			filesMapMutex.Lock()
			f, exists := filesMap[newFile.ID]
			filesMapMutex.Unlock()
			if exists {
				f.LoadingAccounts--
				f.FailAccounts++
				continue
			}
		}

		if skipHeader {
			skipHeader = false
			continue // pass the first line
		}

		task := ProcessTask{Line: line, FileID: newFile.ID}
		taskChannel <- task
	}

	close(taskChannel)

	for _, worker := range workers {
		<-worker.Done
	}

	waitGroup.Wait()

	go func() {
		for {
			filesMapMutex.Lock()
			loadingAccounts := newFile.LoadingAccounts
			filesMapMutex.Unlock()

			if loadingAccounts == 0 {
				loadingAccountsZero <- struct{}{}
				return
			}
		}
	}()

	<-loadingAccountsZero

	updateFileStatus(newFile.ID)

	return newFile.ID, nil
}

func (w *Worker) Run() {
	defer func() {
		w.WaitGroup.Done()
		w.Done <- true

	}()

	for task := range w.TaskQueue {
		line := task.Line
		fileID := task.FileID
		go func() {
			if !utils.ValidateLine(line) {
				log.Printf("Invalid line: %v", line)
				filesMapMutex.Lock()
				f, exists := filesMap[fileID]
				filesMapMutex.Unlock()
				if exists {
					f.FailAccounts++
					f.LoadingAccounts--
				}
				return
			}

			accountEmail := line[2]

			accountsMapMutex.Lock()
			defer accountsMapMutex.Unlock()

			if _, exists := accountsMap[accountEmail]; exists {
				log.Printf("Account already exists: %s", accountEmail)
				filesMapMutex.Lock()
				f, exists := filesMap[fileID]
				filesMapMutex.Unlock()
				if exists {
					f.FailAccounts++
					f.LoadingAccounts--
					return

				}
			}

			log.Printf("Account created: %s", accountEmail)

			account := &models.Account{
				ID:        uuid.New(),
				FirstName: line[0],
				LastName:  line[1],
				Email:     accountEmail,
			}

			accountsMap[accountEmail] = account

			filesMapMutex.Lock()
			defer filesMapMutex.Unlock()

			file, exists := filesMap[fileID]
			if exists {
				file.SuccessAccounts++
				file.LoadingAccounts--
			}
		}()
	}
}

func CreateJobPool(numWorkers int, taskChannel chan ProcessTask, waitGroup *sync.WaitGroup) []*Worker {
	var workers []*Worker

	for i := 0; i < numWorkers; i++ {
		worker := &Worker{
			ID:        i,
			TaskQueue: taskChannel,
			Done:      make(chan bool),
			WaitGroup: waitGroup,
		}
		workers = append(workers, worker)

		waitGroup.Add(1)
		go worker.Run()
	}
	return workers
}
