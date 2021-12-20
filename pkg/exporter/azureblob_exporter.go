package exporter

import (
	"bytes"
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

	ctx := context.Background()

	for key, data := range producer.GetData() {
		appendBlobURL := containerURL.NewAppendBlobURL(fmt.Sprintf("%s/%s/%s", strings.Replace(exporter.creationTime, ":", "-", -1), exporter.hostname, key))

		if _, err := appendBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{}); err != nil {
			storageError, ok := err.(azblob.StorageError)
			if ok {
				switch storageError.ServiceCode() {
				case azblob.ServiceCodeBlobNotFound:
					_, err = appendBlobURL.Create(ctx, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{}, azblob.BlobTagsMap{}, azblob.ClientProvidedKeyOptions{})
					if err != nil {
						return fmt.Errorf("create blob for file %s: %w", key, err)
					}
				default:
					return fmt.Errorf("create blob with storage error: %w", err)
				}
			} else {
				return fmt.Errorf("create blob: %w", err)
			}
		}

		bData := []byte(data)
		start := 0
		size := len(bData)

		for size-start > 0 {
			lengthToWrite := size - start

			if lengthToWrite > azblob.AppendBlobMaxAppendBlockBytes {
				lengthToWrite = azblob.AppendBlobMaxAppendBlockBytes
			}

			w := bData[start : start+lengthToWrite]
			log.Printf("\tAppend blob file: %s (%d bytes), write from %d to %d (%d bytes)", key, size, start, start+lengthToWrite, len(w))

			if _, err = appendBlobURL.AppendBlock(ctx, bytes.NewReader(w), azblob.AppendBlobAccessConditions{}, nil, azblob.ClientProvidedKeyOptions{}); err != nil {
				return fmt.Errorf("append file %s to blob: %w", key, err)
			}

			start += lengthToWrite + 1
		}
	}

	return nil
}

func (exporter *AzureBlobExporter) ExportReader(name string, reader io.ReadSeeker) error {
	containerURL, err := createContainerURL()
	if err != nil {
		return err
	}

	blob := containerURL.NewBlockBlobURL(fmt.Sprintf("%s/%s/%s", strings.Replace(exporter.creationTime, ":", "-", -1), exporter.hostname, name))
	// _, err = blob.Upload(context.Background(), reader, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{}, azblob.DefaultAccessTier, azblob.BlobTagsMap{}, azblob.ClientProvidedKeyOptions{})
	fmt.Printf("Uploading the file with blob name: %s\n", name)
	_, err = azblob.UploadStreamToBlockBlob(context.Background(), reader, blob, azblob.UploadStreamToBlockBlobOptions{})

	return err
}
