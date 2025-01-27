package kubernetes

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ListNamespaces retourne une liste des noms des namespaces.
func ListNamespaces(ctx context.Context, clientset *kubernetes.Clientset) ([]string, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var namespaceNames []string
	for _, ns := range namespaces.Items {
		namespaceNames = append(namespaceNames, ns.Name)
	}

	return namespaceNames, nil
}

// GetFilteredNamespaces retourne une liste des namespaces après avoir appliqué les exclusions.
func GetFilteredNamespaces(ctx context.Context, clientset *kubernetes.Clientset, excludes []string) ([]string, error) {
	// Lister tous les namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	// Construire une map des namespaces exclus pour une recherche rapide
	excludedMap := make(map[string]bool)
	for _, exclude := range excludes {
		excludedMap[exclude] = true
	}

	var filteredNamespaces []string
	for _, ns := range namespaces.Items {
		if !excludedMap[ns.Name] {
			filteredNamespaces = append(filteredNamespaces, ns.Name)
		}
	}

	return filteredNamespaces, nil
}
