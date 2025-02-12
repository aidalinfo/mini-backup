package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ListPVCs retourne une liste des PersistentVolumeClaims pour un namespace donné.
func ListPVCs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list PVCs in namespace %s: %w", namespace, err)
	}
	return pvcs.Items, nil
}

// FindPodUsingPVC recherche un pod qui utilise un PVC donné.
func FindPodUsingPVC(ctx context.Context, clientset *kubernetes.Clientset, namespace, pvcName string) (string, string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pvcName {
				for _, container := range pod.Spec.Containers {
					for _, mount := range container.VolumeMounts {
						if mount.Name == volume.Name {
							return pod.Name, mount.MountPath, nil
						}
					}
				}
			}
		}
	}

	return "", "", fmt.Errorf("no pod found using PVC %s in namespace %s", pvcName, namespace)
}

// CopyPVCData utilise kubectl pour sauvegarder les données d'un PVC dans un fichier tar.
func CopyPVCData(ctx context.Context, kubeConfig, namespace, podName, mountPath, targetPath string) error {
	cmd := exec.CommandContext(
		ctx,
		"kubectl", "exec", "-n", namespace, podName, "--",
		"tar", "cf", "-", mountPath,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeConfig))

	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	defer file.Close()

	var stdErr bytes.Buffer
	cmd.Stdout = file
	cmd.Stderr = &stdErr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute kubectl command: %w. StdErr: %s", err, stdErr.String())
	}

	return nil
}
