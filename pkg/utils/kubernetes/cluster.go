package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"mini-backup/pkg/utils"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterState représente l'état du cluster Kubernetes.
type ClusterState struct {
	Timestamp  string                `json:"timestamp"`
	Namespaces []string              `json:"namespaces"`
	Resources  map[string][]Resource `json:"resources"`
}

// Resource représente une ressource Kubernetes avec ses données.
type Resource struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Kind      string                 `json:"kind"`
	Data      map[string]interface{} `json:"data"`
}

// BackupClusterState sauvegarde l'état du cluster Kubernetes.
func BackupClusterState(ctx context.Context, clientset *kubernetes.Clientset, excludes []string, logger *utils.Logger) (ClusterState, error) {
	logger.Info("Starting cluster state backup")

	namespaces, err := GetFilteredNamespaces(ctx, clientset, excludes)
	if err != nil {
		return ClusterState{}, fmt.Errorf("failed to get namespaces: %w", err)
	}

	state := ClusterState{
		Timestamp:  time.Now().Format(time.RFC3339),
		Namespaces: namespaces,
		Resources:  make(map[string][]Resource),
	}

	// Sauvegarder les ressources namespaced
	for _, namespace := range namespaces {
		logger.Debug(fmt.Sprintf("Processing namespace: %s", namespace))
		saveNamespaceResources(ctx, clientset, namespace, &state, logger)
	}

	// Sauvegarder les ressources cluster-wide
	saveClusterResources(ctx, clientset, &state, logger)

	logger.Info("Cluster state backup completed")
	return state, nil
}

// Sauvegarde des ressources namespaced
func saveNamespaceResources(ctx context.Context, clientset *kubernetes.Clientset, namespace string, state *ClusterState, logger *utils.Logger) {
	resourceTypes := []struct {
		name   string
		listFn func(namespace string) ([]Resource, error)
	}{
		{"persistentvolumeclaims", func(ns string) ([]Resource, error) {
			return listResources(ctx, clientset.CoreV1().PersistentVolumeClaims(ns).List, "persistentvolumeclaims", ns, logger)
		}},
		{"configmaps", func(ns string) ([]Resource, error) {
			return listResources(ctx, clientset.CoreV1().ConfigMaps(ns).List, "configmaps", ns, logger)
		}},
		{"secrets", func(ns string) ([]Resource, error) {
			return listResources(ctx, clientset.CoreV1().Secrets(ns).List, "secrets", ns, logger)
		}},
		{"services", func(ns string) ([]Resource, error) {
			return listResources(ctx, clientset.CoreV1().Services(ns).List, "services", ns, logger)
		}},
		{"deployments", func(ns string) ([]Resource, error) {
			return listResources(ctx, clientset.AppsV1().Deployments(ns).List, "deployments", ns, logger)
		}},
		{"statefulsets", func(ns string) ([]Resource, error) {
			return listResources(ctx, clientset.AppsV1().StatefulSets(ns).List, "statefulsets", ns, logger)
		}},
		{"daemonsets", func(ns string) ([]Resource, error) {
			return listResources(ctx, clientset.AppsV1().DaemonSets(ns).List, "daemonsets", ns, logger)
		}},
		{"pods", func(ns string) ([]Resource, error) {
			return listResources(ctx, clientset.CoreV1().Pods(ns).List, "pods", ns, logger)
		}},
	}

	for _, resType := range resourceTypes {
		resources, err := resType.listFn(namespace)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to list %s in namespace %s: %v", resType.name, namespace, err))
			continue
		}
		state.Resources[resType.name] = append(state.Resources[resType.name], resources...)
	}
}

// Sauvegarde des ressources cluster-wide
func saveClusterResources(ctx context.Context, clientset *kubernetes.Clientset, state *ClusterState, logger *utils.Logger) {
	resourceTypes := []struct {
		name   string
		listFn func() ([]Resource, error)
	}{
		{"persistentvolumes", func() ([]Resource, error) {
			return listResources(ctx, clientset.CoreV1().PersistentVolumes().List, "persistentvolumes", "", logger)
		}},
		{"nodes", func() ([]Resource, error) {
			return listResources(ctx, clientset.CoreV1().Nodes().List, "nodes", "", logger)
		}},
		{"storageclasses", func() ([]Resource, error) {
			return listResources(ctx, clientset.StorageV1().StorageClasses().List, "storageclasses", "", logger)
		}},
	}

	for _, resType := range resourceTypes {
		resources, err := resType.listFn()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to list %s: %v", resType.name, err))
			continue
		}
		state.Resources[resType.name] = append(state.Resources[resType.name], resources...)
	}
}

// Fonction utilitaire pour lister les ressources Kubernetes
func listResources[T any](
	ctx context.Context,
	listFn func(ctx context.Context, opts metav1.ListOptions) (*T, error),
	kind, namespace string,
	logger *utils.Logger,
) ([]Resource, error) {
	// Appeler la fonction pour lister les ressources
	items, err := listFn(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list resources of kind %s: %w", kind, err)
	}

	// Convertir le résultat en JSON pour extraction des champs nécessaires
	var resourceList struct {
		Items []map[string]interface{} `json:"items"`
	}

	data, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal items to JSON: %w", err)
	}

	if err := json.Unmarshal(data, &resourceList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal items from JSON: %w", err)
	}

	// Construire les ressources à partir des données
	var resources []Resource
	for _, item := range resourceList.Items {
		name, _ := item["metadata"].(map[string]interface{})["name"].(string)
		resources = append(resources, Resource{
			Name:      name,
			Namespace: namespace,
			Kind:      kind,
			Data:      item,
		})
	}

	return resources, nil
}
