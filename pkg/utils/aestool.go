package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// readKeyFromFile lit une clé hexadécimale à partir d'un fichier et retourne les bytes
func readKeyFromFile() ([]byte, error) {
	keyHex := GetEnv[string]("AES_KEY")
	// TODO get secret from infisical
	// Décoder la clé hexadécimale
	key, err := hex.DecodeString(string(keyHex))
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors du décodage hexadécimal de la clé : %v", err))
		return nil, fmt.Errorf("erreur lors du décodage hexadécimal de la clé : %v", err)
	}

	// Vérifier la longueur de la clé
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		logger.Error("La clé doit être de 16, 24 ou 32 octets")
		return nil, errors.New("la clé doit être de 16, 24 ou 32 octets")
	}
	logger.Debug(fmt.Sprintf("La clé est de longueur %d octets", len(key)))
	return key, nil
}

// encryptFile chiffre un fichier avec une clé AES
func EncryptFile(inputFile, outputFile string) error {
	// Lire le fichier en clair
	key, err := readKeyFromFile()
	if err != nil {
		return err
	}
	plainText, err := os.ReadFile(inputFile)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la lecture du fichier en clair : %v", err))
		return fmt.Errorf("erreur lors de la lecture du fichier en clair : %v", err)
	}

	// Créer un bloc AES
	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la création du bloc AES : %v", err))
		return fmt.Errorf("erreur lors de la création du bloc AES : %v", err)
	}

	// Créer un nonce
	nonce := make([]byte, 12) // GCM recommande 12 octets pour le nonce
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la génération du nonce : %v", err))
		return fmt.Errorf("erreur lors de la génération du nonce : %v", err)
	}

	// Chiffrer avec AES-GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de la création de GCM : %v", err))
		return fmt.Errorf("erreur lors de la création de GCM : %v", err)
	}

	cipherText := aesGCM.Seal(nil, nonce, plainText, nil)

	// Combiner nonce et texte chiffré
	finalData := append(nonce, cipherText...)

	// Écrire les données chiffrées dans le fichier de sortie
	if err := os.WriteFile(outputFile, finalData, 0644); err != nil {
		logger.Error(fmt.Sprintf("Erreur lors de l'écriture du fichier chiffré : %v", err))
		return fmt.Errorf("erreur lors de l'écriture du fichier chiffré : %v", err)
	}
	logger.Debug(fmt.Sprintf("Fichier chiffré avec succès : %s", outputFile))
	return nil
}

// decryptFile déchiffre un fichier avec une clé AES
func DecryptFile(inputFile, outputFile string) error {
	// Lire le fichier chiffré
	key, err := readKeyFromFile()
	if err != nil {
		return err
	}
	encryptedData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("erreur lors de la lecture du fichier chiffré : %v", err)
	}

	// Créer un bloc AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du bloc AES : %v", err)
	}

	// Extraire le nonce et le texte chiffré
	nonceSize := 12
	if len(encryptedData) < nonceSize {
		return errors.New("données chiffrées trop courtes")
	}

	nonce, cipherText := encryptedData[:nonceSize], encryptedData[nonceSize:]

	// Déchiffrer avec AES-GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de GCM : %v", err)
	}

	plainText, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return fmt.Errorf("erreur lors du déchiffrement : %v", err)
	}

	// Écrire les données en clair dans le fichier de sortie
	if err := os.WriteFile(outputFile, plainText, 0644); err != nil {
		return fmt.Errorf("erreur lors de l'écriture du fichier en clair : %v", err)
	}

	return nil
}
// DecryptBytes déchiffre des données AES-GCM en mémoire
func DecryptBytes(encryptedData []byte) ([]byte, error) {
	key, err := readKeyFromFile()
	if err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("🔐 Début du déchiffrement - Taille chiffrée : %d octets", len(encryptedData)))

	nonceSize := 12
	if len(encryptedData) < nonceSize {
		logger.Error("❌ Données chiffrées trop courtes pour être valides !")
		return nil, errors.New("données chiffrées trop courtes")
	}

	nonce, cipherText := encryptedData[:nonceSize], encryptedData[nonceSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Error(fmt.Sprintf("❌ Erreur lors de la création du bloc AES : %v", err))
		return nil, fmt.Errorf("erreur lors de la création du bloc AES : %v", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		logger.Error(fmt.Sprintf("❌ Erreur lors de la création de GCM : %v", err))
		return nil, fmt.Errorf("erreur lors de la création de GCM : %v", err)
	}

	plainText, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("❌ Erreur lors du déchiffrement : %v", err))
		return nil, fmt.Errorf("erreur lors du déchiffrement : %v", err)
	}

	logger.Info(fmt.Sprintf("🔓 Déchiffrement réussi ! Taille du fichier déchiffré : %d octets", len(plainText)))

	return plainText, nil
}
