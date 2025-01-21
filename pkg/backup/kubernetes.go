package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mini-backup/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// BackupKube sauvegarde toutes les ressources Kubernetes dans un fichier JSON.
func BackupKube(kubeConfigPath, backupDir string) error {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting Kubernetes backup using config: %s", kubeConfigPath))

	// Charger la configuration kubectl
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load kubeconfig: %v", err))
		return err
	}

	// Créer le client Kubernetes
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create Kubernetes client: %v", err))
		return err
	}

	// Initialiser l'état du cluster
	state := ClusterState{
		Timestamp: time.Now().Format(time.RFC3339),
		Resources: make(map[string][]Resource),
	}

	ctx := context.TODO()

	// Sauvegarder les namespaces
	if err := backupNamespaces(ctx, clientset, &state, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to backup namespaces: %v", err))
		return err
	}

	// Sauvegarder les ressources namespaced
	backupNamespacedResources(ctx, clientset, &state, logger)

	// Sauvegarder les ressources au niveau cluster
	backupClusterResources(ctx, clientset, &state, logger)

	// Sauvegarder l'état dans un fichier JSON
	backupFile := filepath.Join(backupDir, fmt.Sprintf("k8s-backup-%s.json", time.Now().Format("2006-01-02-15-04-05")))
	if err := saveToFile(state, backupFile, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to save backup file: %v", err))
		return err
	}

	logger.Info(fmt.Sprintf("Kubernetes backup completed successfully. File saved at: %s", backupFile))
	return nil
}

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

func backupNamespaces(ctx context.Context, clientset *kubernetes.Clientset, state *ClusterState, logger *utils.Logger) error {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error retrieving namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		state.Namespaces = append(state.Namespaces, ns.Name)
	}
	logger.Info(fmt.Sprintf("Namespaces backed up: %v", state.Namespaces))
	return nil
}

// On modifie la signature de saveResources pour utiliser un type générique
func saveResources[T any](ctx context.Context, listFunc func(context.Context, metav1.ListOptions) (*T, error), kind, namespace string, state *ClusterState, logger *utils.Logger) {
	resources, err := listFunc(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(fmt.Sprintf("Error retrieving %s in namespace %s: %v", kind, namespace, err))
		return
	}

	items, err := extractItems(resources)
	if err != nil {
		logger.Error(fmt.Sprintf("Error extracting items for %s: %v", kind, err))
		return
	}

	for _, item := range items {
		resource := Resource{
			Name:      extractName(item),
			Namespace: namespace,
			Kind:      kind,
			Data:      objectToMap(item, logger),
		}
		state.Resources[kind] = append(state.Resources[kind], resource)
	}
	logger.Debug(fmt.Sprintf("Backed up %d %s in namespace %s", len(items), kind, namespace))
}

// Modification de la fonction backupNamespacedResources pour utiliser le type générique
func backupNamespacedResources(ctx context.Context, clientset *kubernetes.Clientset, state *ClusterState, logger *utils.Logger) {
	for _, ns := range state.Namespaces {
		saveResources(ctx, clientset.CoreV1().Pods(ns).List, "pods", ns, state, logger)
		saveResources(ctx, clientset.CoreV1().Services(ns).List, "services", ns, state, logger)
		saveResources(ctx, clientset.CoreV1().ConfigMaps(ns).List, "configmaps", ns, state, logger)
		saveResources(ctx, clientset.CoreV1().Secrets(ns).List, "secrets", ns, state, logger)
		// saveResources(ctx, clientset.CoreV1().PersistentVolumeClaims(ns).List, "persistentvolumeclaims", ns, state, logger)

		saveResources(ctx, clientset.AppsV1().Deployments(ns).List, "deployments", ns, state, logger)
		saveResources(ctx, clientset.AppsV1().StatefulSets(ns).List, "statefulsets", ns, state, logger)
		saveResources(ctx, clientset.AppsV1().DaemonSets(ns).List, "daemonsets", ns, state, logger)
		saveResources(ctx, clientset.BatchV1().Jobs(ns).List, "jobs", ns, state, logger)
		saveResources(ctx, clientset.RbacV1().Roles(ns).List, "roles", ns, state, logger)
		saveResources(ctx, clientset.RbacV1().RoleBindings(ns).List, "rolebindings", ns, state, logger)
	}
}

// Modification de la fonction backupClusterResources pour utiliser le type générique
func backupClusterResources(ctx context.Context, clientset *kubernetes.Clientset, state *ClusterState, logger *utils.Logger) {
	// saveResources(ctx, clientset.CoreV1().Nodes().List, "nodes", "", state, logger)
	saveResources(ctx, clientset.CoreV1().PersistentVolumes().List, "persistentvolumes", "", state, logger)
	// saveResources(ctx, clientset.StorageV1().StorageClasses().List, "storageclasses", "", state, logger)
}

func saveToFile(state ClusterState, backupFile string, logger *utils.Logger) error {
	jsonData, err := json.MarshalIndent(state, "", "    ")
	if err != nil {
		return fmt.Errorf("error serializing state to JSON: %w", err)
	}

	if err := os.WriteFile(backupFile, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing backup file: %w", err)
	}
	return nil
}

func extractItems(obj interface{}) ([]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []interface{} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

func extractName(obj interface{}) string {
	data, _ := json.Marshal(obj)
	var result struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
	}
	_ = json.Unmarshal(data, &result)
	return result.Metadata.Name
}

func objectToMap(obj interface{}, logger *utils.Logger) map[string]interface{} {
	data, err := json.Marshal(obj)
	if err != nil {
		logger.Error(fmt.Sprintf("Error converting object to map: %v", err))
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		logger.Error(fmt.Sprintf("Error deserializing object: %v", err))
		return nil
	}
	return result
}
