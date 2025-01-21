package utils

import (
	"os"

	"github.com/spf13/viper"
)

// GetEnv retourne la valeur typée en fonction de l'utilisation
func GetEnv[T any](key string) T {
	// Configurer Viper pour charger un fichier .env
	if os.Getenv("ENV_PATH") == "" {
		viper.SetConfigFile(".env")
	} else {
		viper.SetConfigFile(os.Getenv("ENV_PATH"))
	}
	viper.AutomaticEnv() // Charger automatiquement les variables d'environnement
	err := viper.ReadInConfig()
	if err != nil {
		// log.Printf("Pas de fichier .env trouvé, utilisation des variables existantes.")
	}
	var zero T // Valeur zéro pour le type attendu (ex: 0 pour int, "" pour string, etc.)

	// Obtenir la valeur brute de Viper
	raw := viper.Get(key)
	if raw == nil {
		// log.Printf("La variable %s n'est pas définie, retour à la valeur par défaut.", key)
		return zero // Retourne la valeur par défaut pour le type attendu
	}

	// Convertir en fonction du type demandé
	val, ok := raw.(T)
	if !ok {
		// log.Printf("Impossible de convertir la variable %s en type attendu, retour à la valeur par défaut.", key)
		return zero // Retourne la valeur par défaut en cas de type non correspond
	}

	return val
}
