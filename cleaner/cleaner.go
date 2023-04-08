package cleaner

import (
	"backubrr/config"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func Cleaner(configPath string) error {
	config, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -config.RetentionDays)
	log.Printf("Cutoff time: %s", cutoff)

	oldBackups := 0

	for _, sourceDir := range config.SourceDirs {
		backupDir := filepath.Join(config.OutputDir, filepath.Base(sourceDir))
		log.Printf("Processing backup directory: %s", backupDir) // Added log message

		err := filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if info.ModTime().Before(cutoff) {
				oldBackups++
				log.Printf("File %s is older than retention period. Deleting...", path)
				if err := os.Remove(path); err != nil {
					return err
				}
			} else {
				log.Printf("File %s is within retention period. Skipping...", path)
			}

			return nil
		})
		if err != nil {
			log.Printf("Error walking the path %q: %v\n", backupDir, err) // Added log message
			return err
		}
	}

	if oldBackups == 0 {
		log.Printf("No old backups found. Cleanup not needed.")
	}

	return nil
}

// Returns true if the directory is empty (contains no files or subdirectories)
func isEmptyDir(path string) bool {
	dir, err := os.Open(path)
	if err != nil {
		return false
	}
	defer dir.Close()

	_, err = dir.Readdir(1)
	if err == nil {
		// Directory is not empty
		return false
	}
	if err == io.EOF {
		// Directory is empty
		return true
	}
	// Error occurred while reading directory
	return false
}
