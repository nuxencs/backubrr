package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"backubrr/config"

	"github.com/briandowns/spinner"
	"github.com/rs/zerolog/log"
)

func CreateBackup(config *config.Config, sourceDir string, passphrase string) error {
	var encryptionKey string
	if passphrase == "" {
		encryptionKey = config.EncryptionKey
	} else {
		encryptionKey = passphrase
	}

	// Print source directory being backed up
	log.Info().Str("sourceDir", sourceDir).Msg("Backing up")

	// Define archive name
	sourceDirBase := filepath.Base(sourceDir)
	archiveName := sourceDirBase + "_" + time.Now().Format("2006-01-02_15-04-05") + ".tar.gz"

	// Update the output directory to include the source directory name
	outputDir := filepath.Join(config.OutputDir, sourceDirBase)

	// Create the output directory if it does not exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
			return err
		}
	}

	// Create destination file for writing
	destFile, err := os.Create(filepath.Join(outputDir, archiveName))
	if err != nil {
		return err
	}

	// Create gzip writer
	gzipWriter := gzip.NewWriter(destFile)

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)

	// Create a new spinner with rotating character set
	spin := spinner.New(spinner.CharSets[50], 100*time.Millisecond)

	// Start the spinner
	spin.Start()

	// Walk through source directory recursively
	filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(path)[0] == '.' {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = path[len(sourceDir)+1:]

		// Write header to tar archive
		if err = tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Open source file for reading
		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		// Copy source file contents to tar archive
		if _, err = io.Copy(tarWriter, sourceFile); err != nil {
			return err
		}

		return nil
	})

	// Stop the spinner
	spin.Stop()

	// Close writers and files
	tarWriter.Close()
	gzipWriter.Close()
	destFile.Close()

	// Encrypt archive using GPG, if encryption key is set
	if encryptionKey != "" {
		encryptedArchiveName := archiveName + ".gpg"
		cmd := exec.Command("gpg", "--batch", "--symmetric", "--cipher-algo", "AES256", "--passphrase", encryptionKey, "--output", filepath.Join(outputDir, encryptedArchiveName), filepath.Join(outputDir, archiveName))

		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			log.Error().Err(err).Msg("Error running GPG command")
			log.Error().Str("GPG output", stderr.String()).Msg("")
			return err
		}

		// Remove unencrypted backup file
		if err := os.Remove(filepath.Join(outputDir, archiveName)); err != nil {
			log.Error().Err(err).Msg("Error removing unencrypted backup file")
		}

		// Print success message for encrypted backup
		message := "Backup created successfully! Encrypted archive saved to " + filepath.Join(config.OutputDir, sourceDirBase, encryptedArchiveName) + "\n\n"
		log.Info().Str("message", message).Msg("")
	} else {
		// Print success message for unencrypted backup
		message := "Backup created successfully! Archive saved to " + filepath.Join(config.OutputDir, sourceDirBase, archiveName) + "\n\n"
		log.Info().Str("message", message).Msg("")
	}

	return nil
}
