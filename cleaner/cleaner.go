package cleaner

import (
	"backubrr/config"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
)

func Cleaner(configPath string) error {
	config, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -config.RetentionDays)

	oldBackups := 0

	for _, sourceDir := range config.SourceDirs {
		backupDir := filepath.Join(config.OutputDir, filepath.Base(sourceDir))
		log.Info().Str("backupDir", backupDir).Msg("Checking backup directory for old backups")

		err := filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if info.ModTime().Before(cutoff) {
				oldBackups++
				log.Info().Str("path", path).Msg("File is older than retention period. Deleting...")
				if err := os.Remove(path); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Error().Str("backupDir", backupDir).Err(err).Msg("Error walking the path")
			return err
		}
	}

	if oldBackups == 0 {
		log.Info().Msg("No old backups found. Cleanup not needed.")
	}

	return nil
}
