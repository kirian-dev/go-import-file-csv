package scripts

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

func GenerateCSV(fileName string, numAccounts int, publicFolder string) error {
	filePath := filepath.Join(publicFolder, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"first_name", "last_name", "email"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for i := 1; i <= numAccounts; i++ {
		firstName := fmt.Sprintf("Test_%d", i)
		lastName := fmt.Sprintf("Test_%d", i)
		email := fmt.Sprintf("test_email_%d@tests.com", i)
		record := []string{firstName, lastName, email}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	fmt.Printf("File created: %s accounts: %d\n", fileName, numAccounts)
	return nil
}
