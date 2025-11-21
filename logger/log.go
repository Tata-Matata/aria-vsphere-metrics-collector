package logger

import (
	"log"
	"os"
	"path/filepath"
)

type Logger struct {
	Dir  string
	file *os.File
}

func Initialize() (*Logger, error) {
	appLog := &Logger{}

	// Determine log directory
	if appLog.Dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Unable to find user home directory: %v", err)
		}

		appLog.Dir = filepath.Join(home, LOG_FOLDER_NAME)

	}

	err := os.MkdirAll(appLog.Dir, 0755)
	if err != nil {
		log.Fatalf("Unable to create log directory: %v", err)
	}

	// Open log file
	logPath := filepath.Join(appLog.Dir, LOG_FILE)

	appLog.file, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	//override default behavior of Go log package to write to file
	log.SetOutput(appLog.file)

	return appLog, nil
}

func (appLog *Logger) Close() {
	if appLog.file != nil {
		appLog.file.Close()
	}
}

func LogError(msg string) {
	log.Println("[ERROR]", msg)
}

func LogInfo(msg string) {
	log.Println("[INFO]", msg)
}

func LogWarn(msg string) {
	log.Println("[WARN]", msg)
}
