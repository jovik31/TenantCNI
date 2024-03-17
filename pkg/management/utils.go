package management

import (
	"bufio"
	"os"
)

var (
	defaultTenantDir = "var/cni/tenants/"
)

func checkIfFileExists() {
}

func createFile(data, filePath string) {
}

func deleteFile() {
}

func LogErrors(text string, filePath string) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		return err
	}

	writer := bufio.NewWriter(f)

	writer.WriteString(text)
	writer.Flush()

	return nil
}


