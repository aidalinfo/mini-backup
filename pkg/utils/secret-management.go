package utils

import (
	"context"
	"fmt"

	infisical "github.com/infisical/go-sdk"
)

func getLogger() *Logger {
	return LoggerFunc()
}
func GetSecret(secretName string, environment string) (string, error) {
	// Récupération des variables d'environnement
	if environment == "" {
		environment = "prod"
	}
	infisicalURL := GetEnv[string]("INFISICAL_URL")
	accessToken := GetEnv[string]("INFISICAL_API_KEY")
	INFISICAL_PROJECT_ID := GetEnv[string]("INFISICAL_PROJECT_ID")
	if infisicalURL == "" {
		infisicalURL = "https://app.infisical.com"
	}

	if accessToken == "" {
		getLogger().Error("L'environnement INFISICAL_API_KEY est manquant")
		return "", fmt.Errorf("l'environnement INFISICAL_API_KEY est manquant")
	}
	getLogger().Info(fmt.Sprintf("Connexion à l'API Infisical : %s", infisicalURL))
	// Initialisation du client Infisical
	client := infisical.NewInfisicalClient(context.Background(), infisical.Config{
		SiteUrl:          infisicalURL,
		AutoTokenRefresh: false,
	})

	// Authentification avec l'API Key
	client.Auth().SetAccessToken(accessToken)

	// Récupération du secret
	secret, err := client.Secrets().Retrieve(infisical.RetrieveSecretOptions{
		SecretKey:   secretName,
		Environment: environment,
		ProjectID:   INFISICAL_PROJECT_ID,
		SecretPath:  "/",
	})
	getLogger().Debug(fmt.Sprintf("Valeur du secret %s : %s", secretName, secret.SecretValue))
	if err != nil {
		getLogger().Error(fmt.Sprintf("Erreur lors de la récupération du secret %s: %v", secretName, err))
		return "", fmt.Errorf("échec de la récupération du secret: %v", err)
	}
	getLogger().Info("Secret récupéré avec succès")
	return secret.SecretValue, nil
}
