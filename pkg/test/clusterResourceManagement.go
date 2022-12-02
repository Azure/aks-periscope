package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func createTestNamespace(clientset *kubernetes.Clientset, name string) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": testingLabelValue,
			},
		},
	}

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating namespace %s: %w", name, err)
	}
	return nil
}

// cleanTestNamespaces deletes all namespaces that have been created for testing purposes.
func cleanTestNamespaces(clientset *kubernetes.Clientset) error {
	namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", testingLabelValue),
	})
	if err != nil {
		return fmt.Errorf("error listing namespaces: %w", err)
	}

	var wg sync.WaitGroup
	var mu = &sync.Mutex{}
	errs := []error{}
	wg.Add(len(namespaceList.Items))
	for _, namespace := range namespaceList.Items {
		go func(name string) {
			defer wg.Done()
			err := clientset.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(namespace.Name)
	}
	wg.Wait()

	if len(errs) > 0 {
		msg := "error cleaning namespaces:"
		for _, err := range errs {
			msg += fmt.Sprintf("\n- %v", err)
		}
		return errors.New(msg)
	}
	return nil
}

func deployPeriscopeServiceAccount(commandRunner *ToolsCommandRunner, kubeConfigFile *os.File, saNamespace string) error {
	command, binds := getDeployPeriscopeServiceAccountCommand(kubeConfigFile.Name(), saNamespace)
	output, err := commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error deploying Periscope service account: %w", err)
	}

	return nil
}

// installMetricsServer installs metrics-server (https://github.com/kubernetes-sigs/metrics-server)
// to the cluster. This is used by the SystemPerf collector.
func installMetricsServer(commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	command, binds := getInstallMetricsServerCommand(kubeConfigFile.Name())
	output, err := commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error installing metrics server: %w", err)
	}

	return nil
}

func installOsm(clientset *kubernetes.Clientset, commandRunner *ToolsCommandRunner, kubeConfigFile *os.File, namespace string) error {
	err := createTestNamespace(clientset, namespace)
	if err != nil {
		return fmt.Errorf("error creating %s namespace: %w", namespace, err)
	}

	command, binds := getInstallOsmCommand(kubeConfigFile.Name(), namespace)
	output, err := commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error running install command for OSM: %w", err)
	}

	return nil
}

func uninstallHelmReleases(commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	command, binds := getUninstallHelmReleasesCommand(kubeConfigFile.Name())
	output, err := commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error running uninstall command for OSM: %w", err)
	}

	return nil
}

func deployOsmApplications(clientset *kubernetes.Clientset, commandRunner *ToolsCommandRunner, kubeConfigFile *os.File, knownNamespaces *KnownNamespaces) error {
	// https://release-v1-1.docs.openservicemesh.io/docs/getting_started/install_apps/
	err := createTestNamespace(clientset, knownNamespaces.OsmBookStore)
	if err != nil {
		return fmt.Errorf("error creating %s namespace: %w", knownNamespaces.OsmBookStore, err)
	}
	err = createTestNamespace(clientset, knownNamespaces.OsmBookBuyer)
	if err != nil {
		return fmt.Errorf("error creating %s namespace: %w", knownNamespaces.OsmBookBuyer, err)
	}
	err = createTestNamespace(clientset, knownNamespaces.OsmBookThief)
	if err != nil {
		return fmt.Errorf("error creating %s namespace: %w", knownNamespaces.OsmBookThief, err)
	}
	err = createTestNamespace(clientset, knownNamespaces.OsmBookWarehouse)
	if err != nil {
		return fmt.Errorf("error creating %s namespace: %w", knownNamespaces.OsmBookWarehouse, err)
	}

	command, binds := getAddOsmNamespacesCommand(kubeConfigFile.Name(), knownNamespaces)
	output, err := commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error adding namespaces to OSM control plane: %w", err)
	}

	command, binds = getDeployOsmAppsCommand(kubeConfigFile.Name(), knownNamespaces)
	output, err = commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error installing applications for OSM: %w", err)
	}

	return nil
}
