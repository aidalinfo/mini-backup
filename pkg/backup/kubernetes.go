package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"mini-backup/pkg/utils"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func BackupPVCData(name string, config utils.Backup, baseVolumesDir string) (string, error) {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting PVC data backup for %s", name))

	// Vérifier la configuration Kubernetes
	if config.Kubernetes == nil {
		return "", fmt.Errorf("kubernetes configuration is missing")
	}

	// Charger la configuration kubectl
	kubeConfigPath := config.Kubernetes.KubeConfig
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load kubeconfig: %v", err))
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create Kubernetes client: %v", err))
		return "", fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	ctx := context.TODO()

	// Liste des namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to list namespaces: %v", err))
		return "", fmt.Errorf("failed to list namespaces: %w", err)
	}

	// Créer une map pour les namespaces exclus pour un accès rapide
	excludedNamespaces := make(map[string]bool)
	for _, exclude := range config.Kubernetes.Volumes.Excludes {
		excludedNamespaces[exclude] = true
	}

	for _, ns := range namespaces.Items {
		namespace := ns.Name

		// Vérifier si le namespace est exclu
		if excludedNamespaces[namespace] {
			logger.Info(fmt.Sprintf("Skipping excluded namespace: %s", namespace))
			continue
		}

		logger.Info(fmt.Sprintf("Processing namespace: %s", namespace))

		// Lister les PVCs dans le namespace
		pvcs, pvcErr := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
		if pvcErr != nil {
			logger.Error(fmt.Sprintf("Failed to list PVCs in namespace %s: %v", namespace, pvcErr))
			continue
		}

		for _, pvc := range pvcs.Items {
			pvcName := pvc.Name
			volumeName := pvc.Spec.VolumeName
			volumeBackupDir := filepath.Join(baseVolumesDir, fmt.Sprintf("%s_%s", pvcName, volumeName))

			logger.Debug(fmt.Sprintf("Creating directory for PVC %s and Volume %s at %s", pvcName, volumeName, volumeBackupDir))

			// Créer un répertoire pour le PVC et le volume
			if err := os.MkdirAll(volumeBackupDir, 0755); err != nil {
				logger.Error(fmt.Sprintf("Failed to create directory for PVC %s: %v", pvcName, err))
				continue
			}

			// Sauvegarder la configuration du PVC
			pvcConfigPath := filepath.Join(volumeBackupDir, "pvc.yaml")
			if err := saveToYAML(pvc, pvcConfigPath); err != nil {
				logger.Error(fmt.Sprintf("Failed to save PVC config for %s: %v", pvcName, err))
				continue
			}

			logger.Debug(fmt.Sprintf("Saved PVC configuration for %s at %s", pvcName, pvcConfigPath))

			// Sauvegarder la configuration du PV
			pv, err := clientset.CoreV1().PersistentVolumes().Get(ctx, volumeName, metav1.GetOptions{})
			if err == nil {
				pvConfigPath := filepath.Join(volumeBackupDir, "pv.yaml")
				if err := saveToYAML(pv, pvConfigPath); err != nil {
					logger.Error(fmt.Sprintf("Failed to save PV config for %s: %v", volumeName, err))
				} else {
					logger.Debug(fmt.Sprintf("Saved PV configuration for %s at %s", volumeName, pvConfigPath))
				}
			} else {
				logger.Error(fmt.Sprintf("Failed to retrieve PV %s: %v", volumeName, err))
			}

			// Trouver un pod qui utilise ce PVC pour sauvegarder les données
			pods, podErr := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			if podErr != nil {
				logger.Error(fmt.Sprintf("Failed to list pods in namespace %s: %v", namespace, podErr))
				continue
			}

			var targetPod, targetMountPath string
			for _, pod := range pods.Items {
				for _, volume := range pod.Spec.Volumes {
					if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pvcName {
						targetPod = pod.Name
						for _, container := range pod.Spec.Containers {
							for _, mount := range container.VolumeMounts {
								if mount.Name == volume.Name {
									targetMountPath = mount.MountPath
									break
								}
							}
						}
					}
				}
			}

			if targetPod == "" || targetMountPath == "" {
				logger.Error(fmt.Sprintf("No pod found using PVC %s in namespace %s", pvcName, namespace))
				continue
			}

			logger.Info(fmt.Sprintf("Found pod %s using PVC %s with mount path %s in namespace %s", targetPod, pvcName, targetMountPath, namespace))

			// Utiliser kubectl exec pour récupérer les données du PVC dans un fichier tar
			tarFilePath := filepath.Join(volumeBackupDir, fmt.Sprintf("%s.tar", pvcName))
			cmd := exec.Command(
				"kubectl", "exec", "-n", namespace, targetPod, "--",
				"tar", "cf", "-", targetMountPath,
			)

			cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath))

			file, err := os.Create(tarFilePath)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to create tar file for PVC %s: %v", pvcName, err))
				continue
			}
			defer file.Close()

			cmd.Stdout = file
			var stdErr bytes.Buffer
			cmd.Stderr = &stdErr

			if err := cmd.Run(); err != nil {
				logger.Error(fmt.Sprintf("kubectl exec failed for PVC %s: %v. StdErr: %s", pvcName, err, stdErr.String()))
				continue
			}

			logger.Info(fmt.Sprintf("Successfully backed up PVC %s to tar file: %s", pvcName, tarFilePath))
		}
	}

	logger.Info(fmt.Sprintf("PVC data backup completed. Backup directory: %s", baseVolumesDir))
	return baseVolumesDir, nil
}

// Fonction utilitaire pour sauvegarder un objet Kubernetes en YAML
func saveToYAML(obj interface{}, filePath string) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("error marshaling object to YAML: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}

func BackupKube(name string, config utils.Backup) ([]string, error) {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting Kubernetes backup for %s", name))

	baseBackupDir := filepath.Join(config.Path.Local, name+"-kubernetes-all-"+time.Now().Format("20060102_150405"))

	// Créer les dossiers principaux
	clusterBackupDir := filepath.Join(baseBackupDir, "Cluster")
	volumesBackupDir := filepath.Join(baseBackupDir, "Volumes")

	if err := os.MkdirAll(clusterBackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cluster backup directory: %w", err)
	}
	if err := os.MkdirAll(volumesBackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create volumes backup directory: %w", err)
	}

	// Sauvegarde des volumes
	if config.Kubernetes.Volumes.Enabled {
		_, err := BackupPVCData(name, config, volumesBackupDir)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to backup PVC data: %v", err))
			return nil, err
		}
		logger.Debug(fmt.Sprintf("Backup of PVC data completed for %s", name))
	}

	// Sauvegarde de l'état du cluster
	if config.Kubernetes.Cluster.Backup == "auto" {
		_, err := BackupClusterState(name, config, clusterBackupDir)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to backup cluster state: %v", err))
			return nil, err
		}
		logger.Debug(fmt.Sprintf("Backup of cluster state completed for %s", name))
	}

	logger.Info(fmt.Sprintf("Kubernetes backup completed for %s. Backup directory: %s", name, baseBackupDir))

	return []string{baseBackupDir}, nil
}

func BackupClusterState(name string, config utils.Backup, clusterBackupDir string) (string, error) {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting cluster state backup for %s", name))

	if config.Kubernetes == nil || config.Kubernetes.Cluster.Backup != "auto" {
		return "", fmt.Errorf("cluster backup is not enabled")
	}

	// Configurer le chemin de sauvegarde
	backupFile := filepath.Join(clusterBackupDir, "cluster-state.json")

	// Initialiser le client Kubernetes
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", config.Kubernetes.KubeConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load kubeconfig: %v", err))
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create Kubernetes client: %v", err))
		return "", fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Sauvegarder l'état du cluster
	state := ClusterState{
		Timestamp: time.Now().Format(time.RFC3339),
		Resources: make(map[string][]Resource),
	}

	ctx := context.TODO()

	// Appeler les fonctions de sauvegarde pour les ressources
	if err := backupNamespaces(ctx, clientset, &state, config, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to backup namespaces: %v", err))
		return "", err
	}

	backupNamespacedResources(ctx, clientset, &state, logger)
	backupClusterResources(ctx, clientset, &state, logger)

	// Enregistrer l'état du cluster dans un fichier JSON
	if err := saveToFile(state, backupFile); err != nil {
		logger.Error(fmt.Sprintf("Failed to save cluster state: %v", err))
		return "", err
	}

	logger.Info(fmt.Sprintf("Cluster state backup completed successfully. File saved at: %s", backupFile))
	return backupFile, nil
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

func backupNamespaces(ctx context.Context, clientset *kubernetes.Clientset, state *ClusterState, config utils.Backup, logger *utils.Logger) error {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error retrieving namespaces: %w", err)
	}

	// Créer une map des namespaces exclus pour une recherche plus rapide
	excludedNS := make(map[string]bool)
	if config.Kubernetes != nil && len(config.Kubernetes.Cluster.Excludes) > 0 {
		for _, ns := range config.Kubernetes.Cluster.Excludes {
			excludedNS[ns] = true
		}
	}

	for _, ns := range namespaces.Items {
		// Vérifier si le namespace doit être exclu
		if !excludedNS[ns.Name] {
			state.Namespaces = append(state.Namespaces, ns.Name)
		} else {
			logger.Info(fmt.Sprintf("Skipping excluded namespace: %s", ns.Name))
		}
	}

	logger.Info(fmt.Sprintf("Namespaces backed up: %v", state.Namespaces))
	return nil
}

func saveResources[T any](ctx context.Context, listFunc func(context.Context, metav1.ListOptions) (*T, error), kind, namespace string, state *ClusterState, logger *utils.Logger) {
	logger.Debug(fmt.Sprintf("Starting backup of %s in namespace %s", kind, namespace))

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

	logger.Debug(fmt.Sprintf("Found %d %s resources in namespace %s", len(items), kind, namespace))

	for _, item := range items {
		name := extractName(item)
		logger.Debug(fmt.Sprintf("Backing up %s/%s in namespace %s", kind, name, namespace))

		resource := Resource{
			Name:      name,
			Namespace: namespace,
			Kind:      kind,
			Data:      objectToMap(item, logger),
		}
		state.Resources[kind] = append(state.Resources[kind], resource)

		// Log les détails de la ressource
		if data := resource.Data; data != nil {
			if metadata, ok := data["metadata"].(map[string]interface{}); ok {
				logger.Debug(fmt.Sprintf("Resource details - Name: %s, Namespace: %s, Labels: %v",
					metadata["name"],
					metadata["namespace"],
					metadata["labels"]))
			}
		}
	}

	logger.Debug(fmt.Sprintf("Successfully backed up %d %s resources in namespace %s",
		len(items), kind, namespace))
}

func backupNamespacedResources(ctx context.Context, clientset *kubernetes.Clientset, state *ClusterState, logger *utils.Logger) {
	// Sauvegarder d'abord les PV (ressources cluster-wide)
	saveResources(ctx, clientset.CoreV1().PersistentVolumes().List, "persistentvolumes", "", state, logger)

	for _, ns := range state.Namespaces {
		// Sauvegarder les PVC immédiatement après les PV
		saveResources(ctx, clientset.CoreV1().PersistentVolumeClaims(ns).List, "persistentvolumeclaims", ns, state, logger)

		// Sauvegarder les ConfigMaps et Secrets en premier car ils sont souvent référencés par d'autres ressources
		saveResources(ctx, clientset.CoreV1().ConfigMaps(ns).List, "configmaps", ns, state, logger)
		saveResources(ctx, clientset.CoreV1().Secrets(ns).List, "secrets", ns, state, logger)

		// Sauvegarder les Services
		saveResources(ctx, clientset.CoreV1().Services(ns).List, "services", ns, state, logger)

		// Sauvegarder les contrôleurs de charge de travail
		saveResources(ctx, clientset.AppsV1().Deployments(ns).List, "deployments", ns, state, logger)
		saveResources(ctx, clientset.AppsV1().StatefulSets(ns).List, "statefulsets", ns, state, logger)
		saveResources(ctx, clientset.AppsV1().DaemonSets(ns).List, "daemonsets", ns, state, logger)
		saveResources(ctx, clientset.BatchV1().Jobs(ns).List, "jobs", ns, state, logger)

		// Sauvegarder les RBAC
		saveResources(ctx, clientset.RbacV1().Roles(ns).List, "roles", ns, state, logger)
		saveResources(ctx, clientset.RbacV1().RoleBindings(ns).List, "rolebindings", ns, state, logger)

		// Sauvegarder les Pods en dernier car ils sont créés par les contrôleurs
		saveResources(ctx, clientset.CoreV1().Pods(ns).List, "pods", ns, state, logger)
	}
}

// Modification de la fonction backupClusterResources pour utiliser le type générique
func backupClusterResources(ctx context.Context, clientset *kubernetes.Clientset, state *ClusterState, logger *utils.Logger) {
	// saveResources(ctx, clientset.CoreV1().Nodes().List, "nodes", "", state, logger)
	saveResources(ctx, clientset.CoreV1().PersistentVolumes().List, "persistentvolumes", "", state, logger)
	// saveResources(ctx, clientset.StorageV1().StorageClasses().List, "storageclasses", "", state, logger)
}

func saveToFile(state ClusterState, backupFile string) error {
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
