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
		return fmt.Errorf("error creating namespace %s: %w", name, err)
	}
	return nil
}

func CleanTestNamespaces(clientset *kubernetes.Clientset) error {
	namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", testingLabel),
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

func InstallMetricsServer(commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	command, binds := GetInstallMetricsServerCommand(kubeConfigFile.Name())
	output, err := commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error installing metrics server: %w", err)
	}

	return nil
}

func InstallOsm(commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	command, binds := GetInstallOsmCommand(kubeConfigFile.Name())
	output, err := commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error running install command for OSM: %w", err)
	}

	return nil
}

func UninstallOsm(commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	command, binds := GetUninstallOsmCommand(kubeConfigFile.Name())
	output, err := commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error running uninstall command for OSM: %w", err)
	}

	return nil
}

func DeployOsmApplications(clientset *kubernetes.Clientset, commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	// https://release-v1-1.docs.openservicemesh.io/docs/getting_started/install_apps/
	err := CreateTestNamespace(clientset, "bookstore")
	if err != nil {
		return fmt.Errorf("error creating bookstore namespace: %w", err)
	}
	err = CreateTestNamespace(clientset, "bookbuyer")
	if err != nil {
		return fmt.Errorf("error creating bookbuyer namespace: %w", err)
	}
	err = CreateTestNamespace(clientset, "bookthief")
	if err != nil {
		return fmt.Errorf("error creating bookthief namespace: %w", err)
	}
	err = CreateTestNamespace(clientset, "bookwarehouse")
	if err != nil {
		return fmt.Errorf("error creating bookwarehouse namespace: %w", err)
	}

	command, binds := GetAddOsmNamespacesCommand(kubeConfigFile.Name())
	output, err := commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error adding namespaces to OSM control plane: %w", err)
	}

	command, binds = GetDeployOsmAppsCommand(kubeConfigFile.Name())
	_, err = commandRunner.Run(command, binds...)
	fmt.Printf("%s\n%s\n\n", command, output)
	if err != nil {
		return fmt.Errorf("error installing applications for OSM: %w", err)
	}

	return nil
}
