package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Compress compresse un fichier ou un dossier donné en gzip et retourne le chemin du fichier compressé.
func Compress(path string) (compressedPath string, err error) {
	// Vérifier si le chemin existe et déterminer s'il s'agit d'un fichier ou d'un dossier
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to access path: %w", err)
	}

	if info.IsDir() {
		// Si c'est un répertoire, créer une archive tar.gz
		compressedPath = path + ".tar.gz"
		err = compressDirectory(path, compressedPath)
		if err != nil {
			return "", fmt.Errorf("failed to compress directory: %w", err)
		}
	} else {
		// Si c'est un fichier, compresser en gzip
		compressedPath = path + ".gz"
		err = compressFile(path, compressedPath)
		if err != nil {
			return "", fmt.Errorf("failed to compress file: %w", err)
		}
	}

	return compressedPath, nil
}

// compressFile compresse un fichier individuel en gzip
func compressFile(filePath, compressedPath string) error {
	// Ouvrir le fichier source
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Créer le fichier compressé
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer compressedFile.Close()

	// Créer un Writer gzip
	writer := gzip.NewWriter(compressedFile)
	defer writer.Close()

	// Copier le contenu du fichier source dans le Writer gzip
	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("failed to compress file: %w", err)
	}

	return nil
}

// compressDirectory compresse un répertoire en tar.gz
func compressDirectory(directoryPath, compressedPath string) error {
	// Créer le fichier .tar.gz
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to create gzip file: %w", err)
	}
	defer compressedFile.Close()

	// Créer un Writer gzip
	gzipWriter := gzip.NewWriter(compressedFile)
	defer gzipWriter.Close()

	// Créer un Writer tar
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Parcourir le répertoire et ajouter les fichiers à l'archive tar
	err = filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		// Calculer le chemin relatif pour conserver l'arborescence
		relativePath, err := filepath.Rel(directoryPath, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// Ignorer le répertoire racine (on ajoute seulement son contenu)
		if relativePath == "." {
			return nil
		}

		// Créer un en-tête tar
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", path, err)
		}
		header.Name = relativePath

		// Écrire l'en-tête dans le Writer tar
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", path, err)
		}

		// Si c'est un fichier, ajouter son contenu au tar
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file %s to tar: %w", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error compressing directory: %w", err)
	}

	return nil
}

// Decompress décompresse un fichier gzip ou tar.gz
func Decompress(compressedPath, outputPath string) (string, error) {
	logger := LoggerFunc()

	// Ouvrir le fichier compressé
	file, err := os.Open(compressedPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to open compressed file: %s : %v", compressedPath, err))
		return "", err
	}
	defer file.Close()

	// Créer un Reader gzip
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create gzip reader for: %s : %v", compressedPath, err))
		return "", err
	}
	defer gzipReader.Close()

	// Vérifier si c'est une archive tar.gz ou un fichier gzip
	if filepath.Ext(compressedPath) == ".gz" && filepath.Ext(strings.TrimSuffix(compressedPath, ".gz")) == ".tar" {
		// Décompresser une archive tar.gz
		tarReader := tar.NewReader(gzipReader)
		logger.Info(fmt.Sprintf("Starting decompression of tar.gz archive: %s", compressedPath))

		// Créer le répertoire de sortie si nécessaire
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			logger.Error(fmt.Sprintf("Failed to create output directory: %s : %v", outputPath, err))
			return "", err
		}

		// Parcourir les fichiers dans l'archive tar
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				// Fin de l'archive
				break
			}
			if err != nil {
				logger.Error(fmt.Sprintf("Error reading tar archive: %s : %v", compressedPath, err))
				return "", err
			}

			// Définir le chemin de sortie
			targetPath := filepath.Join(outputPath, header.Name)

			// Vérifier le type d'entrée
			switch header.Typeflag {
			case tar.TypeDir:
				// Créer un répertoire
				if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
					logger.Error(fmt.Sprintf("Failed to create directory: %s : %v", targetPath, err))
					return "", err
				}
				logger.Debug(fmt.Sprintf("Created directory: %s", targetPath))
			case tar.TypeReg:
				// Créer un fichier
				file, err := os.Create(targetPath)
				if err != nil {
					logger.Error(fmt.Sprintf("Failed to create file: %s : %v", targetPath, err))
					return "", err
				}
				defer file.Close()

				// Copier le contenu dans le fichier
				if _, err := io.Copy(file, tarReader); err != nil {
					logger.Error(fmt.Sprintf("Failed to write file: %s : %v", targetPath, err))
					return "", err
				}
				logger.Debug(fmt.Sprintf("Extracted file: %s", targetPath))
			default:
				logger.Error(fmt.Sprintf("Unsupported tar entry type for: %s", header.Name))
			}
		}

		logger.Info(fmt.Sprintf("Successfully decompressed tar.gz archive: %s", compressedPath))
		return outputPath, nil
	} else {
		// Décompresser un fichier gzip
		logger.Info(fmt.Sprintf("Starting decompression of gzip file: %s", compressedPath))
		decompressedFilePath := strings.TrimSuffix(compressedPath, ".gz")
		outputFile, err := os.Create(decompressedFilePath)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create output file: %s : %v", decompressedFilePath, err))
			return "", err
		}
		defer outputFile.Close()

		// Copier le contenu du gzip dans le fichier
		if _, err := io.Copy(outputFile, gzipReader); err != nil {
			logger.Error(fmt.Sprintf("Failed to write decompressed file: %s : %v", decompressedFilePath, err))
			return "", err
		}
		logger.Info(fmt.Sprintf("Successfully decompressed gzip file: %s to: %s", compressedPath, decompressedFilePath))

		// Vérifier si le fichier décompressé est une archive tar
		if filepath.Ext(decompressedFilePath) == ".tar" {
			logger.Info(fmt.Sprintf("Detected tar archive: %s, decompressing further", decompressedFilePath))
			tarOutputPath := strings.TrimSuffix(decompressedFilePath, ".tar")
			err := DecompressTar(decompressedFilePath, tarOutputPath)
			if err != nil {
				return "", err
			}
			// Supprimer le fichier .tar après extraction
			if err := os.Remove(decompressedFilePath); err != nil {
				logger.Error(fmt.Sprintf("Failed to delete tar file: %s : %v", decompressedFilePath, err))
			}
			return tarOutputPath, nil
		}

		return decompressedFilePath, nil
	}
}

// DecompressTar décompresse une archive tar
func DecompressTar(tarPath, outputPath string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to open tar file: %s : %v", tarPath, err))
		return err
	}
	defer file.Close()

	tarReader := tar.NewReader(file)
	logger.Info(fmt.Sprintf("Starting decompression of tar archive: %s", tarPath))

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error(fmt.Sprintf("Error reading tar archive: %s : %v", tarPath, err))
			return err
		}

		targetPath := filepath.Join(outputPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				logger.Error(fmt.Sprintf("Failed to create directory: %s : %v", targetPath, err))
				return err
			}
		case tar.TypeReg:
			file, err := os.Create(targetPath)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to create file: %s : %v", targetPath, err))
				return err
			}
			defer file.Close()
			if _, err := io.Copy(file, tarReader); err != nil {
				logger.Error(fmt.Sprintf("Failed to write file: %s : %v", targetPath, err))
				return err
			}
		default:
			logger.Error(fmt.Sprintf("Unsupported tar entry type for: %s", header.Name))
		}
	}
	logger.Info(fmt.Sprintf("Successfully decompressed tar archive: %s", tarPath))
	return nil
}
