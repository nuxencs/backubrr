package main

import (
	"backubrr/backup"
	"backubrr/cleaner"
	"backubrr/config"
	"backubrr/notifications"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	version string = "unknown"
	commit  string = "unknown"
	date    string = "unknown"
)

func init() {
	if v := os.Getenv("BACKUBRR_VERSION"); v != "" {
		version = v
	}
	if c := os.Getenv("BACKUBRR_COMMIT"); c != "" {
		commit = c
	}
	if d := os.Getenv("BACKUBRR_DATE"); d != "" {
		date = d
	}
}

func PrintVersion() {
	fmt.Printf("backubrr v%s %s %s\n", version, commit[:7], date)
}

func stringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func listBackupFiles(backupDir string) ([]string, error) {
	files, err := ioutil.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	var fileNames []string
	for _, file := range files {
		if file.Mode().IsRegular() {
			fileNames = append(fileNames, file.Name())
		}
	}

	return fileNames, nil
}

func main() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339, NoColor: false}
	log.Logger = log.Output(output).With().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	flag.Usage = config.PrintHelp
	var configFilePath string
	var backupMessages []string
	var passphrase string
	flag.StringVar(&configFilePath, "config", "config.yaml", "path to config file")
	flag.StringVar(&passphrase, "passphrase", "", "encryption key passphrase")
	flag.Parse()

	if len(os.Args) == 2 && (os.Args[1] == "version" || os.Args[1] == "-v" || os.Args[1] == "--version") {
		PrintVersion()
		return
	}

	// Load configuration from file
	config, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config file, check if the file exists and is valid")
	}

	// Check if encryption key is set correctly
	if config.EncryptionKey != "" && passphrase != "" {
		log.Error().Msg("Encryption key is already set in config. Please remove the --passphrase argument or unset the encryption key in the config file.")
		os.Exit(1)
	}

	// Create destination directory if it doesn't exist
	for _, sourceDir := range config.SourceDirs {
		backupDir := filepath.Join(config.OutputDir, filepath.Base(sourceDir))
		err = os.MkdirAll(backupDir, 0755)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error creating backup directory %s", backupDir)
		}
	}

	for {
		// Create backup for each source directory
		for _, sourceDir := range config.SourceDirs {

			backupDir := filepath.Join(config.OutputDir, filepath.Base(sourceDir))

			// List existing backup files before creating a new backup
			existingBackupFiles, err := listBackupFiles(backupDir)
			if err != nil {
				log.Error().Msgf("Error listing backup files in %s: %s\n", backupDir, err)
				continue
			}

			err = backup.CreateBackup(config, sourceDir, passphrase)
			if err != nil {
				log.Error().Msgf("Error creating backup of %s: %s\n", sourceDir, err)
				continue
			}

			// List backup files after creating a new backup
			newBackupFiles, err := listBackupFiles(backupDir)
			if err != nil {
				log.Error().Msgf("Error listing backup files in %s: %s\n", backupDir, err)
				continue
			}

			// Find newly created backup file
			var newBackupFile string
			for _, file := range newBackupFiles {
				if !stringInSlice(file, existingBackupFiles) {
					newBackupFile = file
					break
				}
			}

			if newBackupFile == "" {
				log.Debug().Msgf("No new backup file found in %s\n", backupDir)
				continue
			}

			var backupMessage string
			if config.EncryptionKey != "" || passphrase != "" {
				backupMessage = fmt.Sprintf("Backup of **`%s`** created successfully! Encrypted archive saved to **`%s`**\n", sourceDir, filepath.Join(backupDir, newBackupFile))
			} else {
				backupMessage = fmt.Sprintf("Backup of **`%s`** created successfully! Archive saved to **`%s`**\n", sourceDir, filepath.Join(backupDir, newBackupFile))
			}
			backupMessage = strings.Replace(backupMessage, os.Getenv("HOME"), "~", -1)
			backupMessages = append(backupMessages, backupMessage)
		}

		// Combine backup messages into a single message
		backupMessage := strings.Join(backupMessages, "")

		// Calculate next backup time
		var nextBackupTime time.Time
		if config.Interval > 0 {
			duration := time.Duration(config.Interval) * time.Hour
			nextBackupTime = time.Now().Add(duration)
		}

		// Create next backup message
		var nextBackupMessage string
		if !nextBackupTime.IsZero() {
			nextBackupMessage = "\nNext backup will run at **`" + nextBackupTime.Format("2006-01-02 15:04:05") + "`**\n"
		}

		// Combine backup message and next scheduled backup message
		combinedMessage := backupMessage + nextBackupMessage

		// Send combined message to Discord
		if config.DiscordWebhookURL != "" {
			if err := notifications.SendToDiscordWebhook(config.DiscordWebhookURL, []string{combinedMessage}); err != nil {
				log.Error().Msgf("Error sending message to Discord:", err)
			}
		}

		// Clean up old backups
		if err := cleaner.Cleaner(configFilePath); err != nil {
			log.Error().Msgf("Error cleaning up old backups:", err)
		}

		// Sleep until the next backup time, if configured
		if config.Interval > 0 {
			if nextBackupTime.IsZero() {
				duration := time.Duration(config.Interval) * time.Hour
				nextBackupTime = time.Now().Add(duration)
			}
			log.Info().Msgf("Next backup will run at %s\n", nextBackupTime.Format("2006-01-02 15:04:05"))
			time.Sleep(time.Until(nextBackupTime))
		} else {
			break
		}
	}
}
