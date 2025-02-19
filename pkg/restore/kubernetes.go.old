package restore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"mini-backup/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterState struct {
	Timestamp  string                `json:"timestamp"`
	Namespaces []string              `json:"namespaces"`
	Resources  map[string][]Resource `json:"resources"`
}

type Resource struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Kind      string                 `json:"kind"`
	Data      map[string]interface{} `json:"data"`
}

var resourceOrder = []string{
	"persistentvolumes",
	"persistentvolumeclaims",
	"configmaps",
	"secrets",
	"services",
	"deployments",
	"statefulsets",
	"daemonsets",
	"roles",
	"rolebindings",
	"pods",
}

func RestoreKube(backupFile string, config utils.Backup, restoreConfig utils.KubernetesRestore) error {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting Kubernetes restore from file: %s", backupFile))

	kubeConfigPath := restoreConfig.KubeConfig

	// Read cluster state file
	clusterStateFile := filepath.Join(backupFile, "Cluster", "cluster-state.json")
	data, err := os.ReadFile(clusterStateFile)
	if err != nil {
		return fmt.Errorf("error reading cluster state file: %w", err)
	}

	// Parse backup data
	var state ClusterState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("error parsing backup data: %w", err)
	}

	// Create Kubernetes clients
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return fmt.Errorf("error building config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return fmt.Errorf("error creating clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(k8sConfig)
	if err != nil {
		return fmt.Errorf("error creating dynamic client: %w", err)
	}

	ctx := context.TODO()

	// Restore namespaces
	namespacesToRestore := determineNamespaces(state.Namespaces, restoreConfig.Cluster, logger)
	if err := restoreNamespaces(ctx, clientset, namespacesToRestore, logger); err != nil {
		logger.Error(fmt.Sprintf("Error restoring namespaces: %v", err))
		return err
	}

	// Define resource GVRs
	gvrMap := defineGVRMap()

	// Restore resources dynamically based on the configuration
	for _, kind := range resourceOrder {
		logger.Info(fmt.Sprintf("Processing resource type: %s", kind))
		resources := filterResources(state.Resources[kind], kind, restoreConfig, namespacesToRestore)

		for _, res := range resources {
			cleanResourceData(res.Data, kind)
			if err := restoreResource(ctx, dynamicClient, res, gvrMap[kind], logger); err != nil {
				logger.Error(fmt.Sprintf("Error restoring %s %s/%s: %v", kind, res.Namespace, res.Name, err))
			}

			// Handle data restoration for PVCs
			if kind == "persistentvolumeclaims" && restoreConfig.Volumes.Full {
				if err := copyPVCData(ctx, clientset, res, kubeConfigPath, backupFile, logger); err != nil {
					logger.Error(fmt.Sprintf("Failed to copy data for PVC %s/%s: %v", res.Namespace, res.Name, err))
				}
			}
		}
	}

	return nil
}

func determineNamespaces(allNamespaces []string, clusterConfig utils.KubernetesCluster, logger *utils.Logger) []string {
	if clusterConfig.Full {
		logger.Info("Restoring all namespaces as per configuration")
		return allNamespaces
	}

	logger.Info("Restoring only specific namespaces as per configuration")
	namespaces := []string{}
	for _, ns := range clusterConfig.Namespaces {
		namespaces = append(namespaces, ns.Name)
	}
	return namespaces
}

func restoreNamespaces(ctx context.Context, clientset *kubernetes.Clientset, namespaces []string, logger *utils.Logger) error {
	for _, ns := range namespaces {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}
		_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			logger.Error(fmt.Sprintf("Error creating namespace %s: %v", ns, err))
		}
	}
	return nil
}

func defineGVRMap() map[string]schema.GroupVersionResource {
	return map[string]schema.GroupVersionResource{
		"configmaps":             {Version: "v1", Resource: "configmaps"},
		"secrets":                {Version: "v1", Resource: "secrets"},
		"services":               {Version: "v1", Resource: "services"},
		"persistentvolumes":      {Version: "v1", Resource: "persistentvolumes"},
		"persistentvolumeclaims": {Version: "v1", Resource: "persistentvolumeclaims"},
		"deployments":            {Group: "apps", Version: "v1", Resource: "deployments"},
		"statefulsets":           {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"daemonsets":             {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"roles":                  {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
		"rolebindings":           {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
	}
}

func filterResources(resources []Resource, kind string, restoreConfig utils.KubernetesRestore, allowedNamespaces []string) []Resource {
	if kind == "persistentvolumes" {
		if restoreConfig.Volumes.Full {
			return resources
		}
		// Filter specific volumes
		return filterVolumes(resources, restoreConfig.Volumes.Namespaces)
	}

	// Filter resources by namespaces
	filtered := []Resource{}
	for _, res := range resources {
		if contains(allowedNamespaces, res.Namespace) {
			filtered = append(filtered, res)
		}
	}
	return filtered
}

func filterVolumes(resources []Resource, volumeNamespaces []utils.Namespace) []Resource {
	filtered := []Resource{}
	for _, nsConfig := range volumeNamespaces {
		for _, volumeName := range nsConfig.Volumes {
			for _, res := range resources {
				if res.Name == volumeName {
					filtered = append(filtered, res)
				}
			}
		}
	}
	return filtered
}

func cleanResourceData(data map[string]interface{}, kind string) {
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		delete(metadata, "uid")
		delete(metadata, "resourceVersion")
		delete(metadata, "generation")
		delete(metadata, "creationTimestamp")
		delete(metadata, "managedFields")
		delete(metadata, "finalizers")

		// Pour les PVC, nettoyer les annotations de binding
		if kind == "persistentvolumeclaims" {
			if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
				delete(annotations, "pv.kubernetes.io/bind-completed")
				delete(annotations, "pv.kubernetes.io/bound-by-controller")
				delete(annotations, "volume.beta.kubernetes.io/storage-provisioner")
				delete(annotations, "volume.kubernetes.io/storage-provisioner")
			}
		}
	}

	delete(data, "status")

	// Nettoyer les volumes dans les specs des pods/déploiements/statefulsets
	if spec, ok := data["spec"].(map[string]interface{}); ok {
		switch kind {
		case "persistentvolumeclaims":
			delete(spec, "volumeName")
			delete(spec, "volumeMode")

		case "persistentvolumes":
			delete(spec, "claimRef")
			spec["persistentVolumeReclaimPolicy"] = "Retain"
			delete(spec, "volumeMode")

		case "pods", "deployments", "statefulsets", "daemonsets":
			// Nettoyer les références aux volumes dans les pods
			cleanPodSpec(spec)

			// Pour les déploiements/statefulsets/daemonsets, il faut aller plus profond
			if template, ok := spec["template"].(map[string]interface{}); ok {
				if podSpec, ok := template["spec"].(map[string]interface{}); ok {
					cleanPodSpec(podSpec)
				}
			}
		}
	}
}

func cleanPodSpec(spec map[string]interface{}) {
	// Nettoyer les volumes au niveau du pod
	if volumes, ok := spec["volumes"].([]interface{}); ok {
		cleanedVolumes := make([]interface{}, 0)
		for _, vol := range volumes {
			if volume, ok := vol.(map[string]interface{}); ok {
				// Nettoyer les références PVC
				if pvc, ok := volume["persistentVolumeClaim"].(map[string]interface{}); ok {
					// Garder uniquement le nom du claim
					claimName := pvc["claimName"]
					cleanedVolume := map[string]interface{}{
						"name": volume["name"],
						"persistentVolumeClaim": map[string]interface{}{
							"claimName": claimName,
						},
					}
					cleanedVolumes = append(cleanedVolumes, cleanedVolume)
				} else {
					// Si ce n'est pas un PVC, garder le volume tel quel
					cleanedVolumes = append(cleanedVolumes, volume)
				}
			}
		}
		spec["volumes"] = cleanedVolumes
	}
	// Nettoyer les volumes montés dans les conteneurs
	if containers, ok := spec["containers"].([]interface{}); ok {
		for _, cont := range containers {
			if container, ok := cont.(map[string]interface{}); ok {
				if volumeMounts, ok := container["volumeMounts"].([]interface{}); ok {
					for i, mount := range volumeMounts {
						if volumeMount, ok := mount.(map[string]interface{}); ok {
							// Garder uniquement le nom et le chemin de montage
							cleanedMount := map[string]interface{}{
								"name":      volumeMount["name"],
								"mountPath": volumeMount["mountPath"],
							}
							if subPath, exists := volumeMount["subPath"]; exists {
								cleanedMount["subPath"] = subPath
							}
							volumeMounts[i] = cleanedMount
						}
					}
				}
			}
		}
	}

	// Faire de même pour les init containers s'ils existent
	if initContainers, ok := spec["initContainers"].([]interface{}); ok {
		for _, cont := range initContainers {
			if container, ok := cont.(map[string]interface{}); ok {
				if volumeMounts, ok := container["volumeMounts"].([]interface{}); ok {
					for i, mount := range volumeMounts {
						if volumeMount, ok := mount.(map[string]interface{}); ok {
							cleanedMount := map[string]interface{}{
								"name":      volumeMount["name"],
								"mountPath": volumeMount["mountPath"],
							}
							if subPath, exists := volumeMount["subPath"]; exists {
								cleanedMount["subPath"] = subPath
							}
							volumeMounts[i] = cleanedMount
						}
					}
				}
			}
		}
	}
}

func restoreResource(ctx context.Context, client dynamic.Interface, res Resource, gvr schema.GroupVersionResource, logger *utils.Logger) error {
	unstructuredObj := &unstructured.Unstructured{
		Object: res.Data,
	}

	var result *unstructured.Unstructured
	var err error

	if gvr.Resource == "persistentvolumes" {
		result, err = client.Resource(gvr).Create(ctx, unstructuredObj, metav1.CreateOptions{})
	} else {
		result, err = client.Resource(gvr).Namespace(res.Namespace).Create(ctx, unstructuredObj, metav1.CreateOptions{})
	}

	if err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info(fmt.Sprintf("Resource %s/%s already exists", res.Kind, res.Name))
			return nil
		}
		return fmt.Errorf("failed to create resource: %w", err)
	}

	logger.Debug(fmt.Sprintf("Restored resource details: %+v", result))
	logger.Info(fmt.Sprintf("Successfully restored %s %s/%s", res.Kind, res.Namespace, res.Name))
	return nil
}

func copyPVCData(ctx context.Context, clientset *kubernetes.Clientset, res Resource, kubeConfigPath, backupFile string, logger *utils.Logger) error {
	logger.Info(fmt.Sprintf("Restoring data for PVC %s/%s", res.Namespace, res.Name))

	// Déterminer le chemin source dynamique pour le PVC
	backupDir := filepath.Join(backupFile, "Volumes") // Chemin relatif depuis le répertoire de décompression
	var sourceTarPath string

	// Rechercher le fichier .tar correspondant au PVC
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() && filepath.HasPrefix(file.Name(), res.Name) {
			sourceTarPath = filepath.Join(backupDir, file.Name(), fmt.Sprintf("%s.tar", res.Name))
			break
		}
	}

	if sourceTarPath == "" {
		return fmt.Errorf("no matching backup found for PVC %s/%s", res.Namespace, res.Name)
	}

	// Définir le pod temporaire et son chemin de montage
	targetPod := "restore-helper"
	targetMountPath := "/mnt/restore"
	volumeName := fmt.Sprintf("%s-volume", res.Name)

	tempPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetPod,
			Namespace: res.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "helper",
					Image: "alpine:latest",
					Command: []string{
						"sh", "-c", "while true; do sleep 30; done",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: targetMountPath,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: res.Name,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	_, err = clientset.CoreV1().Pods(res.Namespace).Create(ctx, tempPod, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create helper pod: %w", err)
	}

	defer clientset.CoreV1().Pods(res.Namespace).Delete(ctx, targetPod, metav1.DeleteOptions{})

	cmd := exec.Command(
		"kubectl", "exec", "-n", res.Namespace, targetPod, "--",
		"sh", "-c", fmt.Sprintf("tar xf - -C %s", targetMountPath),
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath))

	tarFile, err := os.Open(sourceTarPath)
	if err != nil {
		return fmt.Errorf("failed to open tar file for PVC %s: %w", res.Name, err)
	}
	defer tarFile.Close()

	cmd.Stdin = tarFile

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy data to PVC %s/%s: %v, output: %s", res.Namespace, res.Name, err, string(output))
	}

	logger.Info(fmt.Sprintf("Successfully restored data for PVC %s/%s", res.Namespace, res.Name))
	return nil
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
