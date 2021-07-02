package exporter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/Azure/azure-storage-blob-go/azblob"
)

const (
	maxContainerNameLength = 63
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

// GetStorageContainerName get storage container name
func GetStorageContainerName(APIServerFQDN string) (string, error) {
	var containerName string
	var err error
	if utils.IsRunningInAks() {
		containerName, err = GetAKSStorageContainerName(APIServerFQDN)
	} else {
		containerName, err = GetNonAKSStorageContainerName(APIServerFQDN)
	}

	//TODO run a sanitizer over the final chars in the containerName
	return containerName, err
}

//GetNonAKSStorageContainerName get the storage container name for non AKS cluster
func GetNonAKSStorageContainerName(APIServerFQDN string) (string, error) {
	containerName := strings.Replace(APIServerFQDN, ".", "-", -1)

	return containerName, nil
}

//GetAKSStorageContainerName get the storage container name when running on an AKS cluster
func GetAKSStorageContainerName(APIServerFQDN string) (string, error) {
	containerName := strings.Replace(APIServerFQDN, ".", "-", -1)

	//TODO DK: I really dont like the line below, it makes for weird behaviour if e.g. .hcp. or -hcp- is in the fqdn for some reason other than being auto-added by AKS
	length := strings.Index(containerName, "-hcp-")

	if length == -1 {
		maxLength := len(containerName)
		length = int(math.Min(float64(maxLength), float64(maxContainerNameLength)))
	}

	containerName = containerName[:length]
	containerName = strings.TrimRight(containerName, "-")
	return containerName, nil
}

//CreateContainerURL creates the full storage container URL including SAS key
func createContainerURL() (azblob.ContainerURL, error) {
	APIServerFQDN, err := utils.GetAPIServerFQDN()
	if err != nil {
		return azblob.ContainerURL{}, err
	}

	containerName, err := GetStorageContainerName(APIServerFQDN)
	if err != nil {
		return azblob.ContainerURL{}, fmt.Errorf("get StorageContainerName: %+w", err)
	}

	ctx := context.Background()

	pipeline := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT_NAME")
	sasKey := os.Getenv("AZURE_BLOB_SAS_KEY")

	ses := utils.GetStorageEndpointSuffix()
	parsedUrl, err := url.Parse(fmt.Sprintf("https://%s.blob.%s/%s%s", accountName, ses, containerName, sasKey))
	if err != nil {
		return azblob.ContainerURL{}, fmt.Errorf("build blob container url: %w", err)
	}

	containerURL := azblob.NewContainerURL(*parsedUrl, pipeline)

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

		if _, err := appendBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{}); err != nil {
			storageError, ok := err.(azblob.StorageError)
			if ok {
				switch storageError.ServiceCode() {
				case azblob.ServiceCodeBlobNotFound:
					_, err = appendBlobURL.Create(ctx, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})
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

			if _, err = appendBlobURL.AppendBlock(ctx, bytes.NewReader(w), azblob.AppendBlobAccessConditions{}, nil); err != nil {
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
	_, err = blob.Upload(context.Background(), reader, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})

	return err
}
