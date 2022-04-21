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

func CreateTestNamespace(clientset *kubernetes.Clientset, name string) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": testingLabel,
			},
		},
	}

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Error creating namespace %s: %v", name, err)
	}
	return nil
}

func CleanTestNamespaces(clientset *kubernetes.Clientset) error {
	namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", testingLabel),
	})
	if err != nil {
		return fmt.Errorf("Error listing namespaces: %v", err)
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
		msg := "Error cleaning namespaces:"
		for _, err := range errs {
			msg += fmt.Sprintf("\n- %v", err)
		}
		return errors.New(msg)
	}
	return nil
}

func InstallMetricsServer(commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	installMetricsServerCommand, binds := GetInstallMetricsServerCommand(kubeConfigFile.Name())
	_, err := commandRunner.Run(installMetricsServerCommand, binds...)
	if err != nil {
		return fmt.Errorf("Error installing metrics server: %v", err)
	}

	return nil
}

func InstallOsm(commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	installOsmCommand, binds := GetInstallOsmCommand(kubeConfigFile.Name())
	_, err := commandRunner.Run(installOsmCommand, binds...)
	if err != nil {
		return fmt.Errorf("Error running install command for OSM: %v", err)
	}

	return nil
}

func UninstallOsm(commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	uninstallOsmCommand, binds := GetUninstallOsmCommand(kubeConfigFile.Name())
	_, err := commandRunner.Run(uninstallOsmCommand, binds...)
	if err != nil {
		return fmt.Errorf("Error running uninstall command for OSM: %v", err)
	}

	return nil
}

func DeployOsmApplications(clientset *kubernetes.Clientset, commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	// https://release-v1-1.docs.openservicemesh.io/docs/getting_started/install_apps/
	err := CreateTestNamespace(clientset, "bookstore")
	if err != nil {
		return fmt.Errorf("Error creating bookstore namespace: %v", err)
	}
	err = CreateTestNamespace(clientset, "bookbuyer")
	if err != nil {
		return fmt.Errorf("Error creating bookbuyer namespace: %v", err)
	}
	err = CreateTestNamespace(clientset, "bookthief")
	if err != nil {
		return fmt.Errorf("Error creating bookthief namespace: %v", err)
	}
	err = CreateTestNamespace(clientset, "bookwarehouse")
	if err != nil {
		return fmt.Errorf("Error creating bookwarehouse namespace: %v", err)
	}

	addOsmNamespacesCommand, binds := GetAddOsmNamespacesCommand(kubeConfigFile.Name())
	_, err = commandRunner.Run(addOsmNamespacesCommand, binds...)
	if err != nil {
		return fmt.Errorf("Error adding namespaces to OSM control plane: %v", err)
	}

	deployOsmAppsCommand, binds := GetDeployOsmAppsCommand(kubeConfigFile.Name())
	_, err = commandRunner.Run(deployOsmAppsCommand, binds...)
	if err != nil {
		return fmt.Errorf("Error installing applications for OSM: %v", err)
	}

	return nil
}
