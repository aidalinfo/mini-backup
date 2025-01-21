package utils

import (
	"context"
	"fmt"
	"io"
	"time"

	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func Int64Ptr(i int64) *int64 {
	return &i
}

type S3Manager struct {
	Client *s3.Client
	Bucket string
}

type BackupDetails struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// awsCredentialFileCreateFunc génère ou met à jour le fichier ~/.aws/credentials avec les clés fournies
func AwsCredentialFileCreateFunc(accessKey, secretKey string, header string) error {
	// Définir le chemin du fichier credentials
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("erreur lors de la récupération du répertoire personnel : %v", err)
	}
	awsCredentialsPath := filepath.Join(homeDir, ".aws", "credentials")

	// Créer le dossier ~/.aws s'il n'existe pas
	err = os.MkdirAll(filepath.Dir(awsCredentialsPath), 0700)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du dossier .aws : %v", err)
	}

	// Lire le contenu existant du fichier credentials s'il existe
	var existingContent string
	if _, err := os.Stat(awsCredentialsPath); err == nil {
		data, err := os.ReadFile(awsCredentialsPath)
		if err != nil {
			return fmt.Errorf("erreur lors de la lecture du fichier credentials : %v", err)
		}
		existingContent = string(data)
	}

	var sectionHeader string
	if header == "" {
		if GetEnv[string]("GO_ENV") == "dev" {
			sectionHeader = "[dev-backup]"
		} else {
			sectionHeader = "[aidalinfo-backup]"
		}
	} else {
		sectionHeader = header
	}
	if existingContent != "" && containsSection(existingContent, sectionHeader) {
		logger.Info(fmt.Sprintf("La section %s existe déjà dans le fichier credentials. Aucune modification nécessaire.", sectionHeader))
		return nil
	}

	newSection := fmt.Sprintf(`%s
aws_access_key_id = %s
aws_secret_access_key = %s
`, sectionHeader, accessKey, secretKey)

	newContent := existingContent + "\n" + newSection

	// Écrire le contenu mis à jour dans le fichier credentials
	err = os.WriteFile(awsCredentialsPath, []byte(newContent), 0600)
	if err != nil {
		return fmt.Errorf("erreur lors de l'écriture du fichier credentials : %v", err)
	}

	logger.Info(fmt.Sprintf("La section %s a été ajoutée avec succès au fichier credentials : %s", sectionHeader, awsCredentialsPath))
	return nil
}

// containsSection vérifie si une section existe déjà dans le contenu du fichier
func containsSection(content, sectionHeader string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == sectionHeader {
			return true
		}
	}
	return false
}

// NewS3Manager initialise le gestionnaire S3 en utilisant la configuration AWS par défaut
func NewS3Manager(bucket, region, endpoint string, awsprofile string) (*S3Manager, error) {
	// Charger la configuration par défaut depuis les fichiers AWS (credentials et config)
	var profileName string
	if awsprofile == "" {
		if GetEnv[string]("GO_ENV") == "dev" {
			profileName = "dev-backup"
		} else {
			profileName = "[aidalinfo-backup]"
		}
	} else {
		profileName = awsprofile
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region), // Région par défaut
		config.WithSharedConfigProfile(profileName),
	)
	if err != nil {
		return nil, fmt.Errorf("erreur lors du chargement de la configuration AWS : %v", err)
	}
	// Initialiser le client S3 avec le point de terminaison Scaleway
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true      // Mode de chemin d'accès (obligatoire pour Scaleway)
		o.BaseEndpoint = &endpoint // Point de terminaison personnalisé
	})

	return &S3Manager{
		Client: client,
		Bucket: bucket,
	}, nil
}

func (m *S3Manager) ListBackupsApi(prefix string) ([]BackupDetails, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: &m.Bucket,
	}
	if prefix != "" {
		input.Prefix = &prefix
	}
	size := int64(0)
	result, err := m.Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la liste des objets avec le préfixe '%s': %v", prefix, err))
		return nil, fmt.Errorf("erreur lors de la liste des objets : %v", err)
	}

	var backups []BackupDetails
	for _, item := range result.Contents {
		backups = append(backups, BackupDetails{
			Key:          *item.Key,
			Size:         size,
			LastModified: *item.LastModified,
		})
	}

	logger.Info(fmt.Sprintf("Liste des backups détaillée (préfixe: '%s'): %v", prefix, backups))
	return backups, nil
}

// ListBackups liste les objets dans le bucket S3 avec un préfixe optionnel
func (m *S3Manager) ListBackups(prefix string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: &m.Bucket,
	}
	if prefix != "" {
		input.Prefix = &prefix
	}

	result, err := m.Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la liste des objets avec le préfixe '%s': %v", prefix, err))
		return nil, fmt.Errorf("erreur lors de la liste des objets : %v", err)
	}

	var backups []string
	for _, item := range result.Contents {
		backups = append(backups, *item.Key)
	}

	logger.Info(fmt.Sprintf("Liste des backups (préfixe: '%s'): %v", prefix, backups))
	return backups, nil
}

// DownloadFileFromS3 télécharge un fichier depuis S3 vers un chemin local
func (m *S3Manager) Download(s3Path, localPath string) error {
	// Préparer la requête pour télécharger l'objet
	getInput := &s3.GetObjectInput{
		Bucket: &m.Bucket,
		Key:    &s3Path,
	}

	// Obtenir l'objet depuis S3
	objectOutput, err := m.Client.GetObject(context.TODO(), getInput)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du téléchargement de %s depuis S3 : %v", s3Path, err))
		return fmt.Errorf("erreur lors du téléchargement de %s : %v", s3Path, err)
	}
	defer objectOutput.Body.Close()

	// Créer le fichier local
	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la création des répertoires pour %s : %v", localPath, err))
		return err
	}
	localFile, err := os.Create(localPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la création du fichier local %s : %v", localPath, err))
		return err
	}
	defer localFile.Close()

	// Copier le contenu de l'objet dans le fichier local
	_, err = io.Copy(localFile, objectOutput.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la copie du contenu de %s vers %s : %v", s3Path, localPath, err))
		return err
	}

	logger.Info(fmt.Sprintf("Fichier %s téléchargé avec succès vers %s", s3Path, localPath))
	return nil
}

// UploadFileToS3 téléverse un fichier local vers un chemin S3
func (m *S3Manager) Upload(localPath, s3Path string) error {
	// Ouvrir le fichier local
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ouverture du fichier %s : %v", localPath, err)
	}
	defer file.Close()

	// Obtenir la taille du fichier
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("erreur lors de la récupération des informations du fichier %s : %v", localPath, err)
	}

	// Préparer la requête de téléversement
	input := &s3.PutObjectInput{
		Bucket:        &m.Bucket,
		Key:           &s3Path,
		Body:          file,
		ContentLength: Int64Ptr(stat.Size()), // Convertir en *int64
		ContentType:   aws.String("application/octet-stream"),
	}

	// Téléverser le fichier
	_, err = m.Client.PutObject(context.TODO(), input)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la téléversement du fichier %s vers %s : %v", localPath, s3Path, err))
		return fmt.Errorf("erreur lors de l'upload vers S3 (local: %s, s3: %s) : %v", localPath, s3Path, err)
	}

	logger.Info(fmt.Sprintf("Fichier %s téléversé avec succès vers %s", localPath, s3Path))
	return nil
}
func (m *S3Manager) ManageRetention(s3Path string, retentionDays int) error {
	// Calculer la date limite
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	// Lister les objets dans le chemin S3
	input := &s3.ListObjectsV2Input{
		Bucket: &m.Bucket,
		Prefix: &s3Path, // Limiter la liste au chemin spécifié
	}
	result, err := m.Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la liste des objets dans %s : %v", s3Path, err))
		return fmt.Errorf("erreur lors de la liste des objets dans %s : %v", s3Path, err)
	}

	// Parcourir les objets et vérifier leur date de modification
	for _, obj := range result.Contents {
		if strings.HasSuffix(*obj.Key, "/") {
			logger.Debug(fmt.Sprintf("Ignoré : %s c'est un dossier", *obj.Key))
			continue
		}
		if obj.LastModified.Before(cutoffDate) {
			// Supprimer les fichiers obsolètes
			err := m.deleteObject(*obj.Key)
			if err != nil {
				logger.Error(fmt.Sprintf("Erreur lors de la suppression de %s : %v", *obj.Key, err))
			} else {
				logger.Info(fmt.Sprintf("Fichier %s supprimé pour respect de la rétention.", *obj.Key))
			}
		}
	}
	return nil
}

// deleteObject supprime un objet du bucket S3
func (m *S3Manager) deleteObject(key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: &m.Bucket,
		Key:    &key,
	}
	_, err := m.Client.DeleteObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'objet %s : %v", key, err)
	}
	return nil
}

// copyBackupToLocal copie tout le contenu d'un bucket S3 vers un répertoire local
func (m *S3Manager) CopyBackupToLocal(destination string) error {
	// Lister tous les objets dans le bucket
	listInput := &s3.ListObjectsV2Input{
		Bucket: &m.Bucket,
	}

	// Obtenir la liste des objets
	result, err := m.Client.ListObjectsV2(context.TODO(), listInput)
	if err != nil {
		return fmt.Errorf("erreur lors de la liste des objets dans le bucket %s : %v", m.Bucket, err)
	}

	// Parcourir chaque objet dans le bucket
	for _, object := range result.Contents {
		// Calculer le chemin local correspondant
		localPath := filepath.Join(destination, *object.Key)

		// Créer les répertoires nécessaires pour le fichier
		err := os.MkdirAll(filepath.Dir(localPath), 0755)
		if err != nil {
			return fmt.Errorf("erreur lors de la création du répertoire pour %s : %v", localPath, err)
		}

		// Télécharger l'objet
		getInput := &s3.GetObjectInput{
			Bucket: &m.Bucket,
			Key:    object.Key,
		}
		objectOutput, err := m.Client.GetObject(context.TODO(), getInput)
		if err != nil {
			return fmt.Errorf("erreur lors du téléchargement de l'objet %s : %v", *object.Key, err)
		}
		defer objectOutput.Body.Close()

		// Créer le fichier local
		localFile, err := os.Create(localPath)
		if err != nil {
			return fmt.Errorf("erreur lors de la création du fichier local %s : %v", localPath, err)
		}
		defer localFile.Close()

		// Copier le contenu de l'objet dans le fichier local
		_, err = io.Copy(localFile, objectOutput.Body)
		if err != nil {
			return fmt.Errorf("erreur lors de la copie du contenu de %s vers %s : %v", *object.Key, localPath, err)
		}

		logger.Debug(fmt.Sprintf("Fichier %s copié avec succès vers %s", *object.Key, localPath))
	}

	return nil
}

func RstorageManager(name string, config *RStorageConfig) (*S3Manager, error) {
	err := AwsCredentialFileCreateFunc(config.AccessKey, config.SecretKey, name)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la génération du fichier AWS credentials : %v", err))
	}
	// Initialisation du S3Manager
	s3Manager, err := NewS3Manager(config.BucketName, config.Region, config.Endpoint, "")
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de l'initialisation du gestionnaire S3 : %v\n", err))
	}
	logger.Info("S3Manager initialized")
	return s3Manager, nil
}

func ManagerStorageFunc() (*S3Manager, error) {
	// Appeler server config
	config, err := GetConfigServer()
	if err != nil {
		return nil, err
	}
	logger.Debug(fmt.Sprintf("config: %v", config))
	// Appeler `getSecret` pour récupérer les informations nécessaires
	var infisical_environnement string
	if GetEnv[string]("GO_ENV") == "dev" {
		infisical_environnement = "dev"
	} else {
		infisical_environnement = "Production"
	}
	bucketName, err := GetSecret("BUCKET_NAME_PROD", infisical_environnement)
	logger.Debug(fmt.Sprintf("bucketName: %s", bucketName))
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur : %v", err))
	}
	accessKey, err := GetSecret("SCW_ACCESS_BACKUP_ACCESS_KEY", infisical_environnement)
	logger.Debug(fmt.Sprintf("accessKey: %s", accessKey))
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur : %v", err))
	}
	secretKey, err := GetSecret("SCW_ACCESS_BACKUP_SECRET_KEY", infisical_environnement)
	logger.Debug(fmt.Sprintf("secretKey: %s", secretKey))
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur : %v", err))
	}
	region, err := GetSecret("BUCKET_REGION_PROD", infisical_environnement)
	logger.Debug(fmt.Sprintf("region: %s", region))
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur : %v", err))
	}
	endpoint, err := GetSecret("BUCKET_ENDPOINT_PROD", infisical_environnement)
	logger.Debug(fmt.Sprintf("endpoint: %s", endpoint))
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur : %v", err))
	}
	logger.Debug(fmt.Sprintf("Configuration S3 : bucketName: %s, region: %s, endpoint: %s", bucketName, region, endpoint))
	// Générer le fichier credentials
	err = AwsCredentialFileCreateFunc(accessKey, secretKey, "")
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la génération du fichier AWS credentials : %v", err))
	}

	// Initialisation du S3Manager
	s3Manager, err := NewS3Manager(bucketName, region, endpoint, "")
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de l'initialisation du gestionnaire S3 : %v\n", err))
	}
	logger.Info("S3Manager initialized")
	// list, err := s3Manager.ListBackups()
	// if err != nil {
	// 	logger.Error(fmt.Sprintf("Erreur lors de la liste des objets : %v", err))
	// }
	// logger.Debug(fmt.Sprintf("Liste des objets : %v", list))
	return s3Manager, nil
}
