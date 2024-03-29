package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
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
			return io.ReadAll(resp.Body)
		}
	}
}

func Contains(flagsList []string, flag string) bool {
	for _, f := range flagsList {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}

func GetContent(readCloserProvider func() (io.ReadCloser, error)) (string, error) {
	reader, err := readCloserProvider()
	if err != nil {
		return "", err
	}

	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
