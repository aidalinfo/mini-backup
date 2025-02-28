package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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
		sectionHeader = "[" + header + "]"
	}
	if existingContent != "" && containsSection(existingContent, sectionHeader) {
		getLogger().Info(fmt.Sprintf("La section %s existe déjà dans le fichier credentials. Aucune modification nécessaire.", sectionHeader))
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

	getLogger().Info(fmt.Sprintf("La section %s a été ajoutée avec succès au fichier credentials : %s", sectionHeader, awsCredentialsPath))
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
func NewS3Manager(bucket, region, endpoint string, awsprofile string, pathStyle bool) (*S3Manager, error) {
	// Charger la configuration par défaut depuis les fichiers AWS (credentials et config)
	var profileName string
	if awsprofile == "" {
		if GetEnv[string]("GO_ENV") == "dev" {
			profileName = "dev-backup"
		} else {
			profileName = "aidalinfo-backup"
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
		o.UsePathStyle = pathStyle      // Mode de chemin d'accès (obligatoire pour Scaleway)
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
	result, err := m.Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la liste des objets avec le préfixe '%s': %v", prefix, err))
		return nil, fmt.Errorf("erreur lors de la liste des objets : %v", err)
	}

	var backups []BackupDetails
	for _, item := range result.Contents {
		backups = append(backups, BackupDetails{
			Key:          *item.Key,
			Size:         *item.Size,
			LastModified: *item.LastModified,
		})
	}

	getLogger().Debug(fmt.Sprintf("Liste des backups détaillée (préfixe: '%s'): %v", prefix, backups),"[UTILS] [S3MANAGER]")
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
		getLogger().Error(fmt.Sprintf("Erreur lors de la liste des objets avec le préfixe '%s': %v", prefix, err))
		return nil, fmt.Errorf("erreur lors de la liste des objets : %v", err)
	}

	var backups []string
	for _, item := range result.Contents {
		backups = append(backups, *item.Key)
	}

	getLogger().Info(fmt.Sprintf("Liste des backups (préfixe: '%s'): %v", prefix, backups))
	return backups, nil
}
// ListBuckets récupère tous les buckets accessibles avec les credentials
func (m *S3Manager) ListBuckets() ([]string, error) {
	// Récupération de la liste des buckets
	result, err := m.Client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la récupération des buckets : %v", err))
		return nil, fmt.Errorf("erreur lors de la récupération des buckets : %v", err)
	}

	// Extraction des noms des buckets
	buckets := []string{}
	for _, bucket := range result.Buckets {
		buckets = append(buckets, *bucket.Name)
	}

	getLogger().Info(fmt.Sprintf("Liste des buckets S3 récupérée avec succès : %v", buckets))
	return buckets, nil
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
		getLogger().Error(fmt.Sprintf("Erreur lors du téléchargement de %s depuis S3 : %v", s3Path, err))
		return fmt.Errorf("erreur lors du téléchargement de %s : %v", s3Path, err)
	}
	defer objectOutput.Body.Close()

	// Créer le fichier local
	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la création des répertoires pour %s : %v", localPath, err))
		return err
	}
	localFile, err := os.Create(localPath)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la création du fichier local %s : %v", localPath, err))
		return err
	}
	defer localFile.Close()

	// Copier le contenu de l'objet dans le fichier local
	_, err = io.Copy(localFile, objectOutput.Body)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la copie du contenu de %s vers %s : %v", s3Path, localPath, err))
		return err
	}

	getLogger().Info(fmt.Sprintf("Fichier %s téléchargé avec succès vers %s", s3Path, localPath))
	return nil
}

// UploadFileToS3 téléverse un fichier local vers un chemin S3
func (m *S3Manager) Upload(localPath, s3Path string, useGlacier bool) error {
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
	// Déterminer la classe de stockage (STANDARD ou GLACIER)
	var storageClass types.StorageClass = types.StorageClassStandard // Valeur par défaut
	if useGlacier {
		storageClass = types.StorageClassGlacier
	}

	// Préparer la requête de téléversement
	input := &s3.PutObjectInput{
		Bucket:        &m.Bucket,
		Key:           &s3Path,
		Body:          file,
		ContentLength: Int64Ptr(stat.Size()), // Convertir en *int64
		ContentType:   aws.String("application/octet-stream"),
		StorageClass:  storageClass,
	}

	// Téléverser le fichier
	_, err = m.Client.PutObject(context.TODO(), input)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la téléversement du fichier %s vers %s : %v", localPath, s3Path, err))
		return fmt.Errorf("erreur lors de l'upload vers S3 (local: %s, s3: %s) : %v", localPath, s3Path, err)
	}

	getLogger().Info(fmt.Sprintf("Fichier %s téléversé avec succès vers %s", localPath, s3Path))
	return nil
}
func (m *S3Manager) ManageRetention(s3Path string, retentionDays int, useGlacier bool) error {
	// Calculer la date limite
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	// Lister les objets dans le chemin S3
	input := &s3.ListObjectsV2Input{
		Bucket: &m.Bucket,
		Prefix: &s3Path, // Limiter la liste au chemin spécifié
	}
	result, err := m.Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la liste des objets dans %s : %v", s3Path, err))
		return fmt.Errorf("erreur lors de la liste des objets dans %s : %v", s3Path, err)
	}

	// Parcourir les objets et vérifier leur date de modification
	for _, obj := range result.Contents {
		if strings.HasSuffix(*obj.Key, "/") {
			getLogger().Debug(fmt.Sprintf("Ignoré : %s c'est un dossier", *obj.Key))
			continue
		}

		// Récupérer les métadonnées de l'objet pour vérifier sa classe de stockage
		headInput := &s3.HeadObjectInput{
			Bucket: &m.Bucket,
			Key:    obj.Key,
		}
		headResult, err := m.Client.HeadObject(context.TODO(), headInput)
		if err != nil {
			getLogger().Error(fmt.Sprintf("Erreur lors de la récupération des métadonnées de %s : %v", *obj.Key, err))
			continue
		}

		// Vérifier si l'objet est en STANDARD ou GLACIER
		storageClass := headResult.StorageClass
		isGlacierObject := (storageClass == types.StorageClassGlacier || storageClass == types.StorageClassDeepArchive)

		// Si on gère les fichiers GLACIER mais que ce fichier n'est pas en Glacier, on le skip
		if useGlacier && !isGlacierObject {
			getLogger().Debug(fmt.Sprintf("Fichier %s ignoré (pas en Glacier)", *obj.Key))
			continue
		}

		// Si on gère les fichiers STANDARD mais que ce fichier est en Glacier, on le skip
		if !useGlacier && isGlacierObject {
			getLogger().Debug(fmt.Sprintf("Fichier %s ignoré (en Glacier)", *obj.Key))
			continue
		}

		// Vérifier si l'objet doit être supprimé
		if obj.LastModified.Before(cutoffDate) {
			err := m.deleteObject(*obj.Key)
			if err != nil {
				getLogger().Error(fmt.Sprintf("Erreur lors de la suppression de %s : %v", *obj.Key, err))
			} else {
				getLogger().Info(fmt.Sprintf("Fichier %s supprimé pour respect de la rétention.", *obj.Key))
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

		getLogger().Debug(fmt.Sprintf("Fichier %s copié avec succès vers %s", *object.Key, localPath))
	}

	return nil
}

func RstorageManager(name string, config *RStorageConfig) (*S3Manager, error) {
	err := AwsCredentialFileCreateFunc(config.AccessKey, config.SecretKey, name)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la génération du fichier AWS credentials : %v", err))
	}
	// Initialisation du S3Manager
	s3Manager, err := NewS3Manager(config.BucketName, config.Region, config.Endpoint, name, config.PathStyle)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de l'initialisation du gestionnaire S3 : %v\n", err))
	}
	getLogger().Info("S3Manager initialized")
	return s3Manager, nil
}

// DownloadAndDecrypt télécharge un fichier chiffré depuis S3, le déchiffre et retourne son contenu en mémoire
func (m *S3Manager) DownloadAndDecrypt(s3Path string) ([]byte, error) {
	// Préparer la requête pour obtenir l'objet
	getInput := &s3.GetObjectInput{
		Bucket: &m.Bucket,
		Key:    &s3Path,
	}

	// Récupérer l'objet S3
	objectOutput, err := m.Client.GetObject(context.TODO(), getInput)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors du téléchargement de %s depuis S3 : %v", s3Path, err))
		return nil, fmt.Errorf("erreur lors du téléchargement de %s : %v", s3Path, err)
	}
	defer objectOutput.Body.Close()

	// Lire le contenu du fichier téléchargé
	var encryptedData bytes.Buffer
	_, err = io.Copy(&encryptedData, objectOutput.Body)
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la lecture des données chiffrées de %s : %v", s3Path, err))
		return nil, fmt.Errorf("erreur lors de la lecture du fichier chiffré : %v", err)
	}

	// Déchiffrer le fichier en mémoire
	decryptedData, err := DecryptBytes(encryptedData.Bytes())
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors du déchiffrement de %s : %v", s3Path, err))
		return nil, fmt.Errorf("erreur lors du déchiffrement : %v", err)
	}

	getLogger().Info(fmt.Sprintf("Fichier %s téléchargé et déchiffré avec succès", s3Path))
	return decryptedData, nil
}

// GeneratePresignedURL génère une URL signée pour le téléchargement d'un fichier S3
func (m *S3Manager) GeneratePresignedURL(s3Path string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(m.Client)

	req, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(m.Bucket),
		Key:    aws.String(s3Path),
	}, s3.WithPresignExpires(expiration))

	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la génération de l'URL signée pour %s : %v", s3Path, err))
		return "", fmt.Errorf("erreur lors de la génération de l'URL signée : %v", err)
	}

	getLogger().Info(fmt.Sprintf("URL signée générée avec succès pour %s", s3Path))
	return req.URL, nil
}

// DoesBucketExist vérifie si un bucket S3 existe
func (m *S3Manager) DoesBucketExist(bucketName string) (bool, error) {
	_, err := m.Client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: &bucketName,
	})
	if err != nil {
		var notFoundErr *types.NotFound
		if errors.As(err, &notFoundErr) {
			return false, nil // Le bucket n'existe pas
		}
		return false, err // Autre erreur
	}
	return true, nil // Le bucket existe
}

// CreateBucket crée un bucket S3 s'il n'existe pas déjà
func (m *S3Manager) CreateBucket(bucketName string) error {
	// Préparer l'entrée pour créer un bucket
	input := &s3.CreateBucketInput{
		Bucket: &bucketName,
	}

	// Création du bucket
	_, err := m.Client.CreateBucket(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du bucket %s : %v", bucketName, err)
	}

	getLogger().Info(fmt.Sprintf("Bucket %s créé avec succès.", bucketName))
	return nil
}

// UploadEmptyFolder crée un "dossier" vide dans S3
func (m *S3Manager) UploadEmptyFolder(folderPath string) error {
	// Ajouter "/" à la fin pour indiquer un dossier
	if !strings.HasSuffix(folderPath, "/") {
			folderPath += "/"
	}

	input := &s3.PutObjectInput{
			Bucket:      &m.Bucket,
			Key:         &folderPath,
			ContentType: aws.String("application/x-directory"), // MIME type indiquant un dossier
	}

	_, err := m.Client.PutObject(context.TODO(), input)
	if err != nil {
			return fmt.Errorf("erreur lors de la création du dossier %s : %v", folderPath, err)
	}

	getLogger().Info(fmt.Sprintf("Dossier %s créé avec succès dans S3", folderPath))
	return nil
}
