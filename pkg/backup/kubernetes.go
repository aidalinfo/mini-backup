package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mini-backup/pkg/utils"
	"mini-backup/pkg/utils/kubernetes"

	"gopkg.in/yaml.v3"
)

func BackupPVCData(name string, config utils.Backup, baseVolumesDir string) (string, error) {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting PVC data backup for %s", name))

	if config.Kubernetes == nil {
		return "", fmt.Errorf("kubernetes configuration is missing")
	}

	ctx := context.TODO()
	clientset, err := kubernetes.GetKubernetesClient(config.Kubernetes.KubeConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create Kubernetes client: %v", err))
		return "", err
	}

	namespaces, err := kubernetes.GetFilteredNamespaces(ctx, clientset, config.Kubernetes.Volumes.Excludes)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to list namespaces: %v", err))
		return "", err
	}

	for _, namespace := range namespaces {
		logger.Info(fmt.Sprintf("Processing namespace: %s", namespace))

		pvcs, err := kubernetes.ListPVCs(ctx, clientset, namespace)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to list PVCs in namespace %s: %v", namespace, err))
			continue
		}

		for _, pvc := range pvcs {
			volumeBackupDir := filepath.Join(baseVolumesDir, fmt.Sprintf("%s_%s", pvc.Name, pvc.Spec.VolumeName))
			if err := os.MkdirAll(volumeBackupDir, 0755); err != nil {
				logger.Error(fmt.Sprintf("Failed to create directory for PVC %s: %v", pvc.Name, err))
				continue
			}

			if err := saveToYAML(pvc, filepath.Join(volumeBackupDir, "pvc.yaml")); err != nil {
				logger.Error(fmt.Sprintf("Failed to save PVC config for %s: %v", pvc.Name, err))
				continue
			}

			pv, err := kubernetes.GetPV(ctx, clientset, pvc.Spec.VolumeName)
			if err == nil {
				if err := saveToYAML(pv, filepath.Join(volumeBackupDir, "pv.yaml")); err != nil {
					logger.Error(fmt.Sprintf("Failed to save PV config for %s: %v", pvc.Spec.VolumeName, err))
				}
			} else {
				logger.Error(fmt.Sprintf("Failed to retrieve PV %s: %v", pvc.Spec.VolumeName, err))
			}

			targetPod, targetMountPath, err := kubernetes.FindPodUsingPVC(ctx, clientset, namespace, pvc.Name)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to find pod using PVC %s: %v", pvc.Name, err))
				continue
			}

			logger.Info(fmt.Sprintf("Found pod %s using PVC %s with mount path %s", targetPod, pvc.Name, targetMountPath))

			tarFilePath := filepath.Join(volumeBackupDir, fmt.Sprintf("%s.tar", pvc.Name))
			if err := kubernetes.CopyPVCData(ctx, config.Kubernetes.KubeConfig, namespace, targetPod, targetMountPath, tarFilePath); err != nil {
				logger.Error(fmt.Sprintf("Failed to backup PVC data for %s: %v", pvc.Name, err))
				continue
			}

			logger.Info(fmt.Sprintf("Successfully backed up PVC %s to tar file: %s", pvc.Name, tarFilePath))
		}
	}

	logger.Info(fmt.Sprintf("PVC data backup completed. Backup directory: %s", baseVolumesDir))
	return baseVolumesDir, nil
}

func BackupKube(name string, config utils.Backup) ([]string, error) {
	logger := utils.LoggerFunc()
	logger.Info(fmt.Sprintf("Starting Kubernetes backup for %s", name))

	baseBackupDir := filepath.Join(config.Path.Local, name+"-kubernetes-all-"+time.Now().Format("20060102_150405"))
	clusterBackupDir := filepath.Join(baseBackupDir, "Cluster")
	volumesBackupDir := filepath.Join(baseBackupDir, "Volumes")

	if err := os.MkdirAll(clusterBackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cluster backup directory: %w", err)
	}
	if err := os.MkdirAll(volumesBackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create volumes backup directory: %w", err)
	}

	if config.Kubernetes.Volumes.Enabled {
		if _, err := BackupPVCData(name, config, volumesBackupDir); err != nil {
			logger.Error(fmt.Sprintf("Failed to backup PVC data: %v", err))
			return nil, err
		}
		logger.Debug(fmt.Sprintf("Backup of PVC data completed for %s", name))
	}

	if config.Kubernetes.Cluster.Backup == "auto" {
		if _, err := BackupClusterState(name, config, clusterBackupDir); err != nil {
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

	backupFile := filepath.Join(clusterBackupDir, "cluster-state.json")
	ctx := context.TODO()
	clientset, err := kubernetes.GetKubernetesClient(config.Kubernetes.KubeConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create Kubernetes client: %v", err))
		return "", err
	}

	state, err := kubernetes.BackupClusterState(ctx, clientset, config.Kubernetes.Cluster.Excludes, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to backup cluster state: %v", err))
		return "", err
	}

	if err := saveToFile(state, backupFile); err != nil {
		logger.Error(fmt.Sprintf("Failed to save cluster state: %v", err))
		return "", err
	}

	logger.Info(fmt.Sprintf("Cluster state backup completed successfully. File saved at: %s", backupFile))
	return backupFile, nil
}

func saveToFile(state kubernetes.ClusterState, backupFile string) error {
	jsonData, err := json.MarshalIndent(state, "", "    ")
	if err != nil {
		return fmt.Errorf("error serializing state to JSON: %w", err)
	}

	if err := os.WriteFile(backupFile, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing backup file: %w", err)
	}
	return nil
}

func saveToYAML(obj interface{}, filePath string) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("error marshaling object to YAML: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}
