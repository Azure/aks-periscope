package exporter

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/Azure/azure-storage-blob-go/azblob"
)

// AzureBlobExporter defines an Azure Blob Exporter
type AzureBlobExporter struct {
	hostname     string
	creationTime string
}

type StorageKeyType string

const (
	Container StorageKeyType = "Container"
)

var storageKeyTypes = map[string]StorageKeyType{
	"Container": Container,
}

func NewAzureBlobExporter(creationTime, hostname string) *AzureBlobExporter {
	return &AzureBlobExporter{
		hostname:     hostname,
		creationTime: creationTime,
	}
}

func createContainerURL() (azblob.ContainerURL, error) {
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT_NAME")
	sasKey := os.Getenv("AZURE_BLOB_SAS_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER_NAME")
	keyType := os.Getenv("AZURE_STORAGE_SAS_KEY_TYPE")

	if accountName == "" || sasKey == "" || containerName == "" {
		log.Print("Storage Account information were not provided. Export to Azure Storage Account will be skiped.")
		return azblob.ContainerURL{}, nil
	}

	ctx := context.Background()

	pipeline := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})

	ses := utils.GetStorageEndpointSuffix()
	url, err := url.Parse(fmt.Sprintf("https://%s.blob.%s/%s%s", accountName, ses, containerName, sasKey))
	if err != nil {
		return azblob.ContainerURL{}, fmt.Errorf("build blob container url: %w", err)
	}

	containerURL := azblob.NewContainerURL(*url, pipeline)

	if _, ok := storageKeyTypes[keyType]; ok {
		return containerURL, nil
	}

	_, err = containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		storageError, ok := err.(azblob.StorageError)
		if ok {
			switch storageError.ServiceCode() {
			case azblob.ServiceCodeContainerAlreadyExists:
			default:
				return azblob.ContainerURL{}, fmt.Errorf("create container with storage error: %w", err)
			}
		} else {
			return azblob.ContainerURL{}, fmt.Errorf("create container: %w", err)
		}
	}

	return containerURL, nil
}

// Export implements the interface method
func (exporter *AzureBlobExporter) Export(producer interfaces.DataProducer) error {
	containerURL, err := createContainerURL()
	if err != nil {
		return err
	}

	for key, data := range producer.GetData() {
		blobURL := containerURL.NewBlockBlobURL(fmt.Sprintf("%s/%s/%s", strings.Replace(exporter.creationTime, ":", "-", -1), exporter.hostname, key))

		log.Printf("\tAppend blob file: %s (of size %d bytes)", key, len(data))
		if _, err = azblob.UploadStreamToBlockBlob(context.Background(), strings.NewReader(data), blobURL, azblob.UploadStreamToBlockBlobOptions{}); err != nil {
			return fmt.Errorf("append file %s to blob: %w", key, err)
		}
	}

	return nil
}

func (exporter *AzureBlobExporter) ExportReader(name string, reader io.ReadSeeker) error {
	containerURL, err := createContainerURL()
	if err != nil {
		return err
	}

	blobUrl := containerURL.NewBlockBlobURL(fmt.Sprintf("%s/%s/%s", strings.Replace(exporter.creationTime, ":", "-", -1), exporter.hostname, name))
	log.Printf("Uploading the file with blob name: %s\n", name)
	_, err = azblob.UploadStreamToBlockBlob(context.Background(), reader, blobUrl, azblob.UploadStreamToBlockBlobOptions{})

	return err
}
