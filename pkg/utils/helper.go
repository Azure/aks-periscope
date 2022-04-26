package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

const (
	// PublicAzureStorageEndpointSuffix defines default Storage Endpoint Suffix
	PublicAzureStorageEndpointSuffix = "core.windows.net"
	// AzureStackCloudName references the value that will be under the key "cloud" in azure.json if the application is running on Azure Stack Cloud
	// https://kubernetes-sigs.github.io/cloud-provider-azure/install/configs/#azure-stack-configuration -- See this documentation for the well-known cloud name.
	AzureStackCloudName = "AzureStackCloud"
)

// Azure defines Azure configuration
type Azure struct {
	Cloud string `json:"cloud"`
}

// AzureStackCloud defines Azure Stack Cloud configuration
type AzureStackCloud struct {
	StorageEndpointSuffix string `json:"storageEndpointSuffix"`
}

type CommandOutputStreams struct {
	Stdout string
	Stderr string
}

// IsAzureStackCloud returns true if the application is running on Azure Stack Cloud
func IsAzureStackCloud(filePaths *KnownFilePaths) bool {
	azureFile, err := os.ReadFile(filePaths.AzureJson)
	if err != nil {
		return false
	}
	var azure Azure
	if err = json.Unmarshal([]byte(azureFile), &azure); err != nil {
		return false
	}
	cloud := azure.Cloud
	return strings.EqualFold(cloud, AzureStackCloudName)
}

func CopyFile(source, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("unable to open source file %s: %w", source, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", destination, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("error copying data to file %s: %w", destination, err)
	}
	return nil
}

// GetStorageEndpointSuffix returns the SES url from the JSON file as a string
func GetStorageEndpointSuffix(knownFilePaths *KnownFilePaths) string {
	if IsAzureStackCloud(knownFilePaths) {
		ascFile, err := os.ReadFile(knownFilePaths.AzureStackCloudJson)
		if err != nil {
			log.Fatalf("unable to locate %s to extract storage endpoint suffix: %v", knownFilePaths.AzureStackCloudJson, err)
		}
		var azurestackcloud AzureStackCloud
		if err = json.Unmarshal([]byte(ascFile), &azurestackcloud); err != nil {
			log.Fatalf("unable to read %s file: %v", knownFilePaths.AzureStackCloudJson, err)
		}
		return azurestackcloud.StorageEndpointSuffix
	}
	return PublicAzureStorageEndpointSuffix
}

// RunCommandOnHost runs a command on host system
func RunCommandOnHost(command string, arg ...string) (string, error) {
	args := []string{"--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid"}
	args = append(args, "--")
	args = append(args, command)
	args = append(args, arg...)

	cmd := exec.Command("nsenter", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("fail to run command on host: %+v", err)
	}

	return string(out), nil
}

// RunCommandOnContainerWithOutputStreams runs a command on container system and returns both the stdout and stderr output streams
func RunCommandOnContainerWithOutputStreams(command string, arg ...string) (CommandOutputStreams, error) {
	cmd := exec.Command(command, arg...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	outputStreams := CommandOutputStreams{stdout.String(), stderr.String()}

	if err != nil {
		return outputStreams, fmt.Errorf("run command in container: %w", err)
	}

	return outputStreams, nil
}

// RunCommandOnContainer  runs a command on container system and returns the stdout output stream
func RunCommandOnContainer(command string, arg ...string) (string, error) {
	outputStreams, err := RunCommandOnContainerWithOutputStreams(command, arg...)
	return outputStreams.Stdout, err
}

// RunBackgroundCommand starts running a command on a container system in the background and returns its process ID
func RunBackgroundCommand(command string, arg ...string) (int, error) {
	cmd := exec.Command(command, arg...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		return 0, fmt.Errorf("start background command in container exited with message %s: %w", stderr.String(), err)
	}
	return cmd.Process.Pid, nil
}

// Finds and kills a process with a given process ID
func KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process with pid %d to kill: %w", pid, err)
	}
	if err := process.Kill(); err != nil {
		return err
	}
	return nil
}

// Tries to issue an HTTP GET request up to maxRetries times
func GetUrlWithRetries(url string, maxRetries int) ([]byte, error) {
	retry := 1
	for {
		resp, err := http.Get(url)
		if err != nil {
			if retry == maxRetries {
				return nil, fmt.Errorf("max retries reached for request HTTP Get %s: %w", url, err)
			}
			retry++
			time.Sleep(5 * time.Second)
		} else {
			defer resp.Body.Close()
			return ioutil.ReadAll(resp.Body)
		}
	}
}

// GetCreationTimeStamp returns a create timestamp
func GetCreationTimeStamp(config *restclient.Config) (string, error) {
	// Creates the clientset
	creationTimeStamp := ""
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("getting access to K8S failed: %w", err)
	}
	podList, err := GetPods(clientset, "aks-periscope")

	if err != nil {
		return "", err
	}

	// List all the pods similar to kubectl get pods -n <my namespace>
	for _, pod := range podList.Items {
		creationTimeStamp = pod.CreationTimestamp.Format(time.RFC3339Nano)
	}

	return creationTimeStamp, nil
}

// GetResourceList gets a list of all resources of given type in a specified namespace
func GetResourceList(kubeCmds []string, separator string) ([]string, error) {
	outputStreams, err := RunCommandOnContainerWithOutputStreams("kubectl", kubeCmds...)

	if err != nil {
		return nil, err
	}

	resourceList := outputStreams.Stdout
	// If the resource is not found within the cluster, then log a message and do not return any resources.
	if len(resourceList) == 0 {
		return nil, fmt.Errorf("no '%s' resource found in the cluster for given kubectl command", kubeCmds[1])
	}

	return strings.Split(strings.Trim(resourceList, "\""), separator), nil
}

func GetPods(clientset *kubernetes.Clientset, namespace string) (*v1.PodList, error) {
	// Create a pod interface for the given namespace
	podInterface := clientset.CoreV1().Pods(namespace)

	// List the pods in the given namespace
	podList, err := podInterface.List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("getting pods failed: %w", err)
	}

	return podList, nil
}

func Contains(flagsList []string, flag string) bool {
	for _, f := range flagsList {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}
