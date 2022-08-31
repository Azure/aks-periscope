package exporter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/Azure/azure-storage-blob-go/azblob"
)

// AzureBlobExporter defines an Azure Blob Exporter
type AzureBlobExporter struct {
	runtimeInfo    *utils.RuntimeInfo
	knownFilePaths *utils.KnownFilePaths
	containerName  string
}

type StorageKeyType string

const (
	Container StorageKeyType = "Container"
)

var storageKeyTypes = map[string]StorageKeyType{
	"Container": Container,
}

func NewAzureBlobExporter(runtimeInfo *utils.RuntimeInfo, knownFilePaths *utils.KnownFilePaths, containerName string) *AzureBlobExporter {
	return &AzureBlobExporter{
		runtimeInfo:    runtimeInfo,
		knownFilePaths: knownFilePaths,
		containerName:  containerName,
	}
}

func createContainerURL(runtimeInfo *utils.RuntimeInfo, knownFilePaths *utils.KnownFilePaths) (azblob.ContainerURL, error) {
	if runtimeInfo.StorageAccountName == "" || runtimeInfo.StorageSasKey == "" || runtimeInfo.StorageContainerName == "" {
		log.Print("Storage Account information were not provided. Export to Azure Storage Account will be skipped.")
		return azblob.ContainerURL{}, errors.New("Storage not configured.")
	}

	ctx := context.Background()

	pipeline := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})

	ses := utils.GetStorageEndpointSuffix(knownFilePaths)
	url, err := url.Parse(fmt.Sprintf("https://%s.blob.%s/%s%s", runtimeInfo.StorageAccountName, ses, runtimeInfo.StorageContainerName, runtimeInfo.StorageSasKey))
	if err != nil {
		return azblob.ContainerURL{}, fmt.Errorf("build blob container url: %w", err)
	}

	containerURL := azblob.NewContainerURL(*url, pipeline)

	if _, ok := storageKeyTypes[runtimeInfo.StorageSasKeyType]; ok {
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
	containerURL, err := createContainerURL(exporter.runtimeInfo, exporter.knownFilePaths)
	if err != nil {
		return err
	}

	for key, value := range producer.GetData() {
		blobURL := containerURL.NewBlockBlobURL(fmt.Sprintf("%s/%s/%s", exporter.containerName, exporter.runtimeInfo.HostNodeName, key))

		log.Printf("\tAppend blob file: %s (of size %d bytes)", key, value.GetLength())

		err = func() error {
			valueReadCloser, err := value.GetReader()
			if err != nil {
				return err
			}

			defer valueReadCloser.Close()

			_, err = azblob.UploadStreamToBlockBlob(context.Background(), valueReadCloser, blobURL, azblob.UploadStreamToBlockBlobOptions{})
			return err
		}()

		if err != nil {
			return fmt.Errorf("append file %s to blob: %w", key, err)
		}
	}

	return nil
}

func (exporter *AzureBlobExporter) ExportReader(name string, reader io.ReadSeeker) error {
	containerURL, err := createContainerURL(exporter.runtimeInfo, exporter.knownFilePaths)
	if err != nil {
		return err
	}

	blobUrl := containerURL.NewBlockBlobURL(fmt.Sprintf("%s/%s/%s", exporter.containerName, exporter.runtimeInfo.HostNodeName, name))
	log.Printf("Uploading the file with blob name: %s\n", name)
	_, err = azblob.UploadStreamToBlockBlob(context.Background(), reader, blobUrl, azblob.UploadStreamToBlockBlobOptions{})

	return err
}
