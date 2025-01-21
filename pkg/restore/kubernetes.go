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
	"configmaps",
	"secrets",
	"persistentvolumes", // Add this
	"persistentvolumeclaims",
	"services",
	"deployments",
	"statefulsets",
	"daemonsets",
	"roles",        // Add this
	"rolebindings", // Add this
}

func RestoreKube(kubeConfigPath, backupFile string) error {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting Kubernetes restore from file: %s", backupFile))

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
		resources := state.Resources[kind]
		for _, res := range resources {
			cleanResourceData(res.Data, kind)
			if err := restoreResource(ctx, dynamicClient, res, gvrMap[kind], logger); err != nil {
				logger.Error(fmt.Sprintf("Error restoring %s %s/%s: %v", kind, res.Namespace, res.Name, err))
				continue
			}
		}
	}

	logger.Info("Kubernetes restore completed successfully")
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
	// Clean metadata
	metadata, ok := data["metadata"].(map[string]interface{})
	if ok {
		delete(metadata, "resourceVersion")
		delete(metadata, "uid")
		delete(metadata, "creationTimestamp")
		delete(metadata, "generation")
		delete(metadata, "selfLink")
		delete(metadata, "managedFields")
		delete(metadata, "ownerReferences")
		delete(metadata, "finalizers")
	}

	// Remove status
	delete(data, "status")

	// Clean specific resource types
	switch kind {
	case "persistentvolumeclaims":
		cleanPVCData(data)
	case "persistentvolumes":
		cleanPVData(data)
	case "services":
		cleanServiceData(data)
	}
}

func cleanPVCData(data map[string]interface{}) {
	spec, ok := data["spec"].(map[string]interface{})
	if !ok {
		return
	}
	delete(spec, "volumeName")
}

func cleanServiceData(data map[string]interface{}) {
	spec, ok := data["spec"].(map[string]interface{})
	if !ok {
		return
	}
	delete(spec, "clusterIP")
	delete(spec, "clusterIPs")
}

func restoreResource(ctx context.Context, client dynamic.Interface, res Resource, gvr schema.GroupVersionResource, logger *utils.Logger) error {
	unstructuredObj := &unstructured.Unstructured{
		Object: res.Data,
	}

	_, err := client.Resource(gvr).Namespace(res.Namespace).Create(ctx, unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info(fmt.Sprintf("Resource %s/%s already exists", res.Kind, res.Name))
			return nil
		}
		return err
	}

	logger.Info(fmt.Sprintf("Successfully restored %s %s/%s", res.Kind, res.Namespace, res.Name))
	return nil
}

func cleanPVData(data map[string]interface{}) {
	spec, ok := data["spec"].(map[string]interface{})
	if !ok {
		return
	}
	// Remove fields that should not be present during creation
	delete(spec, "claimRef")
}
