package restore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

func RestoreKube(backupFile string, config utils.Backup) error {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting Kubernetes restore from file: %s", backupFile))

	// Vérifier la configuration Kubernetes
	if config.Kubernetes == nil {
		return fmt.Errorf("kubernetes configuration is missing")
	}

	kubeConfigPath := config.Kubernetes.KubeConfig

	// Read backup file
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("error reading backup file: %w", err)
	}

	// Parse backup data
	var state ClusterState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("error parsing backup data: %w", err)
	}

	// Create kubernetes clients
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

	// Restore namespaces first
	if err := restoreNamespaces(ctx, clientset, state.Namespaces, logger); err != nil {
		logger.Error(fmt.Sprintf("Error restoring namespaces: %v", err))
	}

	// Define resource GVRs
	gvrMap := map[string]schema.GroupVersionResource{
		"configmaps":             {Version: "v1", Resource: "configmaps"},
		"secrets":                {Version: "v1", Resource: "secrets"},
		"services":               {Version: "v1", Resource: "services"},
		"persistentvolumes":      {Version: "v1", Resource: "persistentvolumes"}, // Add this
		"persistentvolumeclaims": {Version: "v1", Resource: "persistentvolumeclaims"},
		"deployments":            {Group: "apps", Version: "v1", Resource: "deployments"},
		"statefulsets":           {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"daemonsets":             {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"roles":                  {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},        // Add this
		"rolebindings":           {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}, // Add this
	}

	// Restore resources in order
	for _, kind := range resourceOrder {
		logger.Debug(fmt.Sprintf("Starting restoration of resource type: %s", kind))
		resources := state.Resources[kind]
		logger.Debug(fmt.Sprintf("Found %d resources of type %s to restore", len(resources), kind))

		// Vérifier si le GVR existe pour ce type
		gvr, exists := gvrMap[kind]
		if !exists {
			logger.Error(fmt.Sprintf("No GVR mapping found for resource type: %s", kind))
			continue
		}

		for _, res := range resources {
			logger.Debug(fmt.Sprintf("Cleaning and restoring %s: %s/%s", kind, res.Namespace, res.Name))
			cleanResourceData(res.Data, kind)

			// Pour les PV, le namespace doit être vide
			namespace := res.Namespace
			if kind == "persistentvolumes" {
				namespace = ""
				logger.Debug(fmt.Sprintf("Processing PV: %s", res.Name))
			}

			if err := restoreResource(ctx, dynamicClient, res, gvr, logger); err != nil {
				logger.Error(fmt.Sprintf("Error restoring %s %s/%s: %v", kind, namespace, res.Name, err))
				// Log plus de détails sur l'erreur
				logger.Debug(fmt.Sprintf("Resource data that failed: %+v", res.Data))
				continue
			}
		}
		logger.Info(fmt.Sprintf("Completed restoration of resource type: %s", kind))
	}

	return nil
}

func restoreNamespaces(ctx context.Context, clientset *kubernetes.Clientset, namespaces []string, logger *utils.Logger) error {
	for _, ns := range namespaces {
		if ns == "default" || ns == "kube-system" || ns == "kube-public" || ns == "kube-node-lease" {
			continue
		}

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
func cleanResourceData(data map[string]interface{}, kind string) {
	metadata, ok := data["metadata"].(map[string]interface{})
	if ok {
		// Nettoyer les champs générés
		delete(metadata, "uid")
		delete(metadata, "resourceVersion")
		delete(metadata, "generation")
		delete(metadata, "creationTimestamp")
		delete(metadata, "managedFields")
		delete(metadata, "finalizers")
	}

	// Supprimer le status
	delete(data, "status")

	if kind == "persistentvolumes" {
		if spec, ok := data["spec"].(map[string]interface{}); ok {
			// Nettoyer les attributs CSI spécifiques
			if csi, hasCsi := spec["csi"].(map[string]interface{}); hasCsi {
				if attrs, hasAttrs := csi["volumeAttributes"].(map[string]interface{}); hasAttrs {
					// Supprimer l'identité du provisionneur
					delete(attrs, "storage.kubernetes.io/csiProvisionerIdentity")
				}
			}

			// Nettoyer claimRef
			if claimRef, hasClaimRef := spec["claimRef"].(map[string]interface{}); hasClaimRef {
				// Garder seulement les informations essentielles
				delete(claimRef, "uid")
				delete(claimRef, "resourceVersion")
			}

			// S'assurer que la politique de réclamation est Retain
			spec["persistentVolumeReclaimPolicy"] = "Retain"
		}
	}

	logger.Debug(fmt.Sprintf("Cleaned %s data: %+v", kind, data))
}
func restoreResource(ctx context.Context, client dynamic.Interface, res Resource, gvr schema.GroupVersionResource, logger *utils.Logger) error {
	unstructuredObj := &unstructured.Unstructured{
		Object: res.Data,
	}

	logger.Debug(fmt.Sprintf("Attempting to restore resource: %s/%s of type %s", res.Namespace, res.Name, res.Kind))

	var result *unstructured.Unstructured
	var err error

	if res.Kind == "persistentvolumes" {
		// Les PV sont des ressources cluster-wide
		result, err = client.Resource(gvr).Create(ctx, unstructuredObj, metav1.CreateOptions{})
	} else {
		result, err = client.Resource(gvr).Namespace(res.Namespace).Create(ctx, unstructuredObj, metav1.CreateOptions{})
	}

	if err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info(fmt.Sprintf("Resource %s/%s already exists", res.Kind, res.Name))
			return nil
		}
		logger.Debug(fmt.Sprintf("Error details for %s/%s: %v", res.Kind, res.Name, err))
		return fmt.Errorf("failed to create resource: %w", err)
	}

	logger.Debug(fmt.Sprintf("Restored resource details: %v", result.Object))
	logger.Info(fmt.Sprintf("Successfully restored %s %s/%s", res.Kind, res.Namespace, res.Name))
	return nil
}
