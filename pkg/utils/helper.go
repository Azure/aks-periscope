package utils

import (
	"io/ioutil"
	"strings"
)

// GetHostName get host name
func GetHostName() string {
	hostname, _ := ioutil.ReadFile("/etc/hostname")
	return strings.TrimSuffix(string(hostname), "\n")
}

// GetAzureBlobLogin get azure blob login info
func GetAzureBlobLogin() (string, string) {
	accountName, _ := ioutil.ReadFile("/etc/azure-blob/accountName")
	accountKey, _ := ioutil.ReadFile("/etc/azure-blob/accountKey")
	return strings.TrimSuffix(string(accountName), "\n"), strings.TrimSuffix(string(accountKey), "\n")
}
