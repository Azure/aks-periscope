package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
func IsAzureStackCloud() bool {
	azureFile, err := RunCommandOnHost("cat", "/etc/kubernetes/azure.json")
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

// CopyFileFromHost saves the specified source file to the destination
func CopyFileFromHost(source, destination string) error {
	sourceFile, err := RunCommandOnHost("cat", source)
	if err != nil {
		return fmt.Errorf("unable to retrieve source content: %w", err)
	}
	if err = WriteToFile(destination, sourceFile); err != nil {
		return fmt.Errorf("unable to write source file to destination: %w", err)
	}
	return nil
}

// GetStorageEndpointSuffix returns the SES url from the JSON file as a string
func GetStorageEndpointSuffix() string {
	if IsAzureStackCloud() {
		ascFile, err := RunCommandOnHost("cat", "/etc/kubernetes/azurestackcloud.json")
		if err != nil {
			log.Fatalf("unable to locate azurestackcloud.json to extract storage endpoint suffix: %v", err)
		}
		var azurestackcloud AzureStackCloud
		if err = json.Unmarshal([]byte(ascFile), &azurestackcloud); err != nil {
			log.Fatalf("unable to read azurestackcloud.json file: %v", err)
		}
		return azurestackcloud.StorageEndpointSuffix
	}
	return PublicAzureStorageEndpointSuffix
}

// GetHostName get host name
func GetHostName() (string, error) {
	hostname, err := RunCommandOnHost("cat", "/etc/hostname")
	if err != nil {
		return "", fmt.Errorf("Fail to get host name: %+v", err)
	}

	return strings.TrimSuffix(string(hostname), "\n"), nil
}

// GetAPIServerFQDN gets the API Server FQDN from the kubeconfig file
func GetAPIServerFQDN() (string, error) {
	output, err := RunCommandOnHost("cat", "/var/lib/kubelet/kubeconfig")

	if err != nil {
		return "", fmt.Errorf("Can't open kubeconfig file: %+v", err)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		index := strings.Index(line, "server: ")
		if index >= 0 {
			fqdn := line[index+len("server: "):]
			fqdn = strings.Replace(fqdn, "https://", "", -1)
			fqdn = strings.Replace(fqdn, ":443", "", -1)
			return fqdn, nil
		}
	}

	return "", errors.New("Could not find server definitions in kubeconfig")
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
		return "", fmt.Errorf("Fail to run command on host: %+v", err)
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
		return outputStreams, fmt.Errorf("Fail to run command in container: %s", fmt.Sprint(err)+": "+stderr.String())
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
		return 0, fmt.Errorf("Start background command in container exited with message %s: %w", stderr.String(), err)
	}
	return cmd.Process.Pid, nil
}

// Finds and kills a process with a given process ID
func KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("Find process with pid %d to kill: %w", pid, err)
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
				return nil, fmt.Errorf("Max retries reached for request HTTP Get %s: %w", url, err)
			}
			retry++
			time.Sleep(5 * time.Second)
		} else {
			defer resp.Body.Close()
			return ioutil.ReadAll(resp.Body)
		}
	}
}

// WriteToFile writes data to a file
func WriteToFile(fileName string, data string) error {
	if err := os.MkdirAll(filepath.Dir(fileName), os.ModePerm); err != nil {
		return fmt.Errorf("Fail to create path directories for file %s: %w", fileName, err)
	}
	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("Fail to create file %s: %+v", fileName, err)
	}
	defer f.Close()

	_, err = f.Write([]byte(data))
	if err != nil {
		return fmt.Errorf("Fail to write data to file %s: %+v", fileName, err)
	}

	return nil
}

// CreateCollectorDir creates a working dir for a collector
func CreateCollectorDir(name string) (string, error) {
	hostName, err := GetHostName()
	if err != nil {
		return "", err
	}

	creationTimeStamp, err := GetCreationTimeStamp()
	if err != nil {
		return "", err
	}

	rootPath := filepath.Join("/aks-periscope", strings.Replace(creationTimeStamp, ":", "-", -1), hostName, "collector", name)
	err = os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("Fail to create dir %s: %+v", rootPath, err)
	}

	return rootPath, nil
}

// CreateDiagnosticDir creates a working dir for diagnostic
func CreateDiagnosticDir() (string, error) {
	hostName, err := GetHostName()
	if err != nil {
		return "", err
	}

	creationTimeStamp, err := GetCreationTimeStamp()
	if err != nil {
		return "", err
	}

	rootPath := filepath.Join("/aks-periscope", strings.Replace(creationTimeStamp, ":", "-", -1), hostName, "diagnoser")
	err = os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("Fail to create dir %s: %+v", rootPath, err)
	}

	return rootPath, nil
}

// CreateKubeConfigFromServiceAccount creates kubeconfig based on creds in service account
func CreateKubeConfigFromServiceAccount() error {
	token, err := RunCommandOnContainer("cat", "/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return err
	}

	_, err = RunCommandOnContainer("kubectl", "config", "set-credentials", "aks-periscope-service-account", "--token="+token)
	if err != nil {
		return err
	}

	_, err = RunCommandOnContainer("kubectl", "config", "set-cluster", "aks-periscope-cluster", "--server=https://kubernetes.default.svc.cluster.local:443", "--certificate-authority=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return err
	}

	_, err = RunCommandOnContainer("kubectl", "config", "set-context", "aks-periscope-context", "--user=aks-periscope-service-account", "--cluster=aks-periscope-cluster")
	if err != nil {
		return err
	}

	_, err = RunCommandOnContainer("kubectl", "config", "use-context", "aks-periscope-context")
	if err != nil {
		return err
	}

	return nil
}

// GetCreationTimeStamp returns a create timestamp
func GetCreationTimeStamp() (string, error) {
	creationTimeStamp, err := RunCommandOnContainer("kubectl", "get", "pods", "--all-namespaces", "-l", "app=aks-periscope", "-o", "jsonpath=\"{.items[0].metadata.creationTimestamp}\"")
	if err != nil {
		return "", err
	}

	return creationTimeStamp[1 : len(creationTimeStamp)-1], nil
}

// WriteToCRD writes diagnostic data to CRD
func WriteToCRD(fileName string, key string) error {
	hostName, err := GetHostName()
	if err != nil {
		return err
	}

	crdName := "aks-periscope-diagnostic" + "-" + hostName

	jsonBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	patchContent := fmt.Sprintf("{\"spec\":{%q:%q}}", key, string(jsonBytes))

	_, err = RunCommandOnContainer("kubectl", "-n", "aks-periscope", "patch", "apd", crdName, "-p", patchContent, "--type=merge")
	if err != nil {
		return err
	}

	return nil
}

// CreateCRD creates a CRD object
func CreateCRD() error {
	hostName, err := GetHostName()
	if err != nil {
		return err
	}

	crdName := "aks-periscope-diagnostic" + "-" + hostName

	if err = writeDiagnosticCRD(crdName); err != nil {
		return err
	}

	_, err = RunCommandOnContainer("kubectl", "apply", "-f", "aks-periscope-diagnostic-crd.yaml")
	if err != nil {
		return err
	}

	return nil
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
		return nil, fmt.Errorf("No '%s' resource found in the cluster for given kubectl command", kubeCmds[1])
	}

	return strings.Split(strings.Trim(resourceList, "\""), separator), nil
}

func writeDiagnosticCRD(crdName string) error {
	f, err := os.Create("aks-periscope-diagnostic-crd.yaml")
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("apiVersion: \"aks-periscope.azure.github.com/v1\"\n")
	if err != nil {
		return err
	}

	_, err = f.WriteString("kind: Diagnostic\n")
	if err != nil {
		return err
	}

	_, err = f.WriteString("metadata:\n")
	if err != nil {
		return err
	}

	_, err = f.WriteString("  name: " + crdName + "\n")
	if err != nil {
		return err
	}

	_, err = f.WriteString("  namespace: aks-periscope\n")
	if err != nil {
		return err
	}

	_, err = f.WriteString("spec:\n")
	if err != nil {
		return err
	}

	_, err = f.WriteString("  networkconfig: \"\"\n")
	if err != nil {
		return err
	}

	_, err = f.WriteString("  networkoutbound: \"\"\n")
	if err != nil {
		return err
	}

	return nil
}
