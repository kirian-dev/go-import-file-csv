package services

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kirian-dev/go-import-file-csv/internal/models"
	"github.com/kirian-dev/go-import-file-csv/internal/utils"

	"github.com/google/uuid"
)

type ProcessTask struct {
	Line   []string
	FileID uuid.UUID
}
type Semaphore struct {
	C chan struct{}
}

func (s *Semaphore) Acquire() {
	s.C <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.C
}

const (
	uploadFolder = "./public/uploads"
	bufferSize   = 10
	gCount       = 4
)

var (
	accountsMap      = make(map[string]*models.Account)
	filesMap         = make(map[uuid.UUID]*models.File)
	accountsMapMutex sync.Mutex
	filesMapMutex    sync.Mutex
)

func GetFileByID(id uuid.UUID) (*models.File, error) {
	filesMapMutex.Lock()
	f, exists := filesMap[id]
	filesMapMutex.Unlock()
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

	if err := saveFile(file, tempFilePath); err != nil {
		return nil, err
	}

	rows, err := utils.CountLinesInFile(tempFilePath)
	if err != nil {
		return nil, err
	}

	id, err := processFile(tempFilePath, tempFileName, rows, gCount)
	if err != nil {
		return nil, err
	}

	return &models.FileResponse{
		Name: tempFileName,
		ID:   id,
	}, nil
}

func createSemaphore(gCount int) *Semaphore {
	return &Semaphore{
		C: make(chan struct{}, gCount),
	}
}

func saveFile(file *multipart.FileHeader, filePath string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	return err
}

func processFile(filePath, tempFileName string, rows, gCount int) (uuid.UUID, error) {
	newFile := createNewFile(tempFileName, rows)

	sem := createSemaphore(gCount)

	taskChannel := make(chan ProcessTask, bufferSize)
	defer close(taskChannel)

	reader, err := openCSVFile(filePath)
	if err != nil {
		return uuid.Nil, err
	}
	var wg sync.WaitGroup
	go func() {
		skipHeader := true
		for {
			line, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				handleReadError(err, newFile)
			}

			if skipHeader {
				skipHeader = false
				continue // pass first line
			}

			sem.Acquire()
			wg.Add(1)
			go func(line []string, fileID uuid.UUID) {
				defer sem.Release()
				defer wg.Done()
				processTask(line, fileID)
			}(line, newFile.ID)
		}

		wg.Wait()

		updateFileStatus(newFile.ID)
	}()

	return newFile.ID, nil
}
func createNewFile(tempFileName string, rows int) *models.File {
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

	return newFile
}

func openCSVFile(filePath string) (*csv.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	return csv.NewReader(file), nil
}

func handleReadError(err error, newFile *models.File) {
	log.Printf("Error reading line: %v", err)
	filesMapMutex.Lock()
	f, exists := filesMap[newFile.ID]
	filesMapMutex.Unlock()
	if exists {
		f.LoadingAccounts--
		f.FailAccounts++
	}
}

func updateFileStatus(id uuid.UUID) {
	filesMapMutex.Lock()
	f, exists := filesMap[id]
	filesMapMutex.Unlock()

	if exists && f.LoadingAccounts == 0 {
		f.Status = models.SuccessStatus
		f.EndTime = time.Now()
	}
}

func processTask(line []string, fileID uuid.UUID) {
	if !utils.ValidateLine(line) {
		handleInvalidLine(line, fileID)
		return
	}

	accountEmail := line[2]

	accountsMapMutex.Lock()
	defer accountsMapMutex.Unlock()

	if _, exists := accountsMap[accountEmail]; exists {
		handleAccountExists(accountEmail, fileID)
		return
	}

	createAccount(line, accountEmail, fileID)
}

func handleInvalidLine(line []string, fileID uuid.UUID) {
	log.Printf("Invalid line: %v", line)
	filesMapMutex.Lock()
	f, exists := filesMap[fileID]
	filesMapMutex.Unlock()
	if exists {
		f.FailAccounts++
		f.LoadingAccounts--
	}
}

func handleAccountExists(accountEmail string, fileID uuid.UUID) {
	log.Printf("Account already exists: %s", accountEmail)
	filesMapMutex.Lock()
	f, exists := filesMap[fileID]
	filesMapMutex.Unlock()
	if exists {
		f.FailAccounts++
		f.LoadingAccounts--
	}
}

func createAccount(line []string, accountEmail string, fileID uuid.UUID) {
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

	f, exists := filesMap[fileID]
	if exists {
		f.SuccessAccounts++
		f.LoadingAccounts--
	}
}
