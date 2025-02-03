package restore

import (
	"context"
	"fmt"
	"mini-backup/pkg/utils"
	k8sutils "mini-backup/pkg/utils/kubernetes"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/kubernetes"
)

// RestoreKube restaure les volumes Kubernetes en fonction de la configuration donnée.
func RestoreKube(result string, backupConfig utils.Backup, kubeRestoreConfig utils.KubernetesRestore) error {
	logger := utils.LoggerFunc()
	logger.Info("Starting Kubernetes restoration process")

	kubeConfigPath := kubeRestoreConfig.KubeConfig
	fullRestore := kubeRestoreConfig.Volumes.Full
	pvcsToRestore := kubeRestoreConfig.Volumes.PVCs

	ctx := context.TODO()
	clientset, err := k8sutils.GetKubernetesClient(kubeConfigPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create Kubernetes client: %v", err))
		return err
	}

	if fullRestore {
		logger.Info("Full volume restoration is enabled")
		namespaces, err := k8sutils.ListNamespaces(ctx, clientset)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to list namespaces: %v", err))
			return err
		}

		for _, namespace := range namespaces {
			logger.Info(fmt.Sprintf("Restoring volumes for namespace: %s", namespace))
			if err := restoreNamespacePVCs(ctx, clientset, namespace, result, kubeConfigPath, logger); err != nil {
				logger.Error(fmt.Sprintf("Failed to restore volumes for namespace %s: %v", namespace, err))
				continue
			}
		}
	} else {
		logger.Info("Targeted PVC restoration is enabled")
		for _, pvcName := range pvcsToRestore {
			logger.Info(fmt.Sprintf("Restoring PVC: %s", pvcName))
			if err := restorePVC(ctx, clientset, pvcName, result, kubeConfigPath, logger); err != nil {
				logger.Error(fmt.Sprintf("Failed to restore PVC %s: %v", pvcName, err))
			}
		}
	}

	logger.Info("Kubernetes restoration process completed")
	return nil
}

func restoreNamespacePVCs(ctx context.Context, clientset *kubernetes.Clientset, namespace, result, kubeConfigPath string, logger *utils.Logger) error {
	logger.Info(fmt.Sprintf("Restoring PVCs in namespace: %s", namespace))

	pvcs, err := k8sutils.ListPVCs(ctx, clientset, namespace)
	if err != nil {
		return fmt.Errorf("failed to list PVCs in namespace %s: %w", namespace, err)
	}

	for _, pvc := range pvcs {
		if err := restorePVC(ctx, clientset, pvc.Name, result, kubeConfigPath, logger); err != nil {
			logger.Error(fmt.Sprintf("Failed to restore PVC %s in namespace %s: %v", pvc.Name, namespace, err))
			continue
		}
	}

	return nil
}

func restorePVC(ctx context.Context, clientset *kubernetes.Clientset, pvcName, result, kubeConfigPath string, logger *utils.Logger) error {
	namespace, err := getNamespaceForPVC(ctx, clientset, pvcName)
	if err != nil {
		return fmt.Errorf("failed to get namespace for PVC %s: %w", pvcName, err)
	}

	// Recherche des dossiers correspondant à `*_pvcName`
	var matchingDirs []string
	baseBackupDir := filepath.Join(result, "Volumes")
	err = filepath.Walk(baseBackupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Vérifier si le dossier correspond au schéma `*_pvcName`
		if info.IsDir() && strings.Contains(info.Name(), pvcName) {
			matchingDirs = append(matchingDirs, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error searching for backup directories for PVC %s: %w", pvcName, err)
	}

	if len(matchingDirs) == 0 {
		return fmt.Errorf("no backup directories found for PVC %s", pvcName)
	}

	// Parcourir chaque dossier correspondant
	for _, backupDir := range matchingDirs {
		var tarFilePath string

		// Recherche dynamique du fichier `.tar` dans le répertoire
		err = filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".tar") {
				tarFilePath = path
			}
			return nil
		})
		if err != nil {
			logger.Error(fmt.Sprintf("Error while searching for tar file in %s: %v", backupDir, err))
			continue
		}

		if tarFilePath == "" {
			logger.Error(fmt.Sprintf("No tar file found in %s for PVC %s", backupDir, pvcName))
			continue
		}

		// Identifier un Pod existant attaché au PVC
		targetPod, mountPath, err := k8sutils.FindPodUsingPVC(ctx, clientset, namespace, pvcName)
		if err != nil || targetPod == "" {
			logger.Error(fmt.Sprintf("No pod found using PVC %s in namespace %s", pvcName, namespace))
			return fmt.Errorf("failed to find an existing pod for PVC %s", pvcName)
		}

		logger.Info(fmt.Sprintf("Restoring data to PVC %s using pod %s and mount path %s", pvcName, targetPod, mountPath))

		// Restaurer les données avec kubectl
		if err := k8sutils.RestorePVCData(ctx, kubeConfigPath, namespace, targetPod, mountPath, tarFilePath); err != nil {
			logger.Error(fmt.Sprintf("Failed to restore data to PVC %s: %v", pvcName, err))
			continue
		}

		logger.Info(fmt.Sprintf("Successfully restored PVC %s from directory %s", pvcName, backupDir))
	}

	return nil
}

func getNamespaceForPVC(ctx context.Context, clientset *kubernetes.Clientset, pvcNameOrUID string) (string, error) {
	namespaces, err := k8sutils.ListNamespaces(ctx, clientset)
	if err != nil {
		return "", fmt.Errorf("failed to list namespaces: %w", err)
	}

	for _, namespace := range namespaces {
		pvcs, err := k8sutils.ListPVCs(ctx, clientset, namespace)
		if err != nil {
			continue
		}
		for _, pvc := range pvcs {
			if pvc.Name == pvcNameOrUID || string(pvc.UID) == pvcNameOrUID {
				return namespace, nil
			}
		}
	}

	return "", fmt.Errorf("namespace for PVC %s not found", pvcNameOrUID)
}
