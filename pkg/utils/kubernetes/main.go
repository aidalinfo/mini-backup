package kubernetes

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubernetesClient crée et retourne un client Kubernetes à partir d'un fichier kubeconfig.
func GetKubernetesClient(kubeConfigPath string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if kubeConfigPath != "" {
		// Charger la configuration à partir du fichier kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeConfigPath, err)
		}
	} else {
		// Charger la configuration à partir de l'environnement (cluster)
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load in-cluster config: %w", err)
		}
	}

	// Créer un clientset Kubernetes
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return clientset, nil
}
