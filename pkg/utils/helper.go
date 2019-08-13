package utils

import (
	"io/ioutil"
	"os/exec"
	"strings"
)

// GetHostName get host name
func GetHostName() string {
	hostname, _ := RunCommandOnHost("cat", "/etc/hostname")
	return strings.TrimSuffix(string(hostname), "\n")
}

// GetAzureBlobCredential get azure blob access info
func GetAzureBlobCredential() (string, string) {
	accountName, _ := ioutil.ReadFile("/etc/azure-blob/accountName")
	sasKey, _ := ioutil.ReadFile("/etc/azure-blob/sasKey")
	return strings.TrimSuffix(string(accountName), "\n"), strings.TrimSuffix(string(sasKey), "\n")
}

// RunCommandOnHost runs a command on host system
func RunCommandOnHost(command string, arg ...string) (string, error) {
	args := []string{"--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid"}
	args = append(args, "--")
	args = append(args, command)
	args = append(args, arg...)

	cmd := exec.Command("nsenter", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
