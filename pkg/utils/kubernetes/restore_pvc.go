package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// RestorePVCData utilise kubectl pour restaurer les données d'un fichier .tar dans un PVC.
func RestorePVCData(ctx context.Context, kubeConfig, namespace, podName, mountPath, tarFilePath string) error {
	// Vérifier que le fichier tar existe
	if _, err := os.Stat(tarFilePath); os.IsNotExist(err) {
		return fmt.Errorf("tar file not found at %s", tarFilePath)
	}

	// Ouvrir le fichier tar
	file, err := os.Open(tarFilePath)
	if err != nil {
		return fmt.Errorf("failed to open tar file %s: %w", tarFilePath, err)
	}
	defer file.Close()

	// Commande kubectl pour exécuter la restauration
	cmd := exec.CommandContext(
		ctx,
		"kubectl", "exec", "-n", namespace, podName, "--",
		"tar", "xf", "-", "-C", mountPath,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeConfig))
	cmd.Stdin = file

	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	// Exécuter la commande
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restore data with kubectl: %w. StdErr: %s", err, stdErr.String())
	}

	return nil
}
