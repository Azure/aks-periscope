package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

// RunCommandOnContainer runs a command on container system
func RunCommandOnContainer(command string, arg ...string) (string, error) {
	cmd := exec.Command(command, arg...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Fail to run command in container: %+v", err)
	}

	return string(out), nil
}

// WriteToFile writes data to a file
func WriteToFile(fileName string, data string) error {
	f, err := os.Create(fileName)
	defer f.Close()
	if err != nil {
		return fmt.Errorf("Fail to create file %s: %+v", fileName, err)
	}

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

	rootPath := filepath.Join("/aks-periscope/", hostName, "metrics", name)
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

	rootPath := filepath.Join("/aks-periscope/", hostName, "diagnostic")
	err = os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("Fail to create dir %s: %+v", rootPath, err)
	}

	return rootPath, nil
}
