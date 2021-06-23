package exporter

import (
	"bytes"
	"context"
	"fmt"
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
	BaseExporter
}

var _ interfaces.Exporter = &AzureBlobExporter{}

// NewAzureBlobExporter is a constructor
func NewAzureBlobExporter() *AzureBlobExporter {
	return &AzureBlobExporter{
		BaseExporter: BaseExporter{
			exporterType: AzureBlob,
		},
	}
}

// GetStorageContainerName get storage container name
func (exporter *AzureBlobExporter) GetStorageContainerName(APIServerFQDN string) (string, error) {
	var containerName string
	var err error
	if utils.IsKubernetesInDocker() {
		containerName, err = exporter.GetKubernetesInDockerStorageContainerName(APIServerFQDN)
	} else {
		containerName, err = exporter.GetNonKINDStorageContainerName(APIServerFQDN)
	}

	//TODO run a sanitizer over the final chars in the containerName
	return containerName, err
}

func (exporter *AzureBlobExporter) GetKubernetesInDockerStorageContainerName(APIServerFQDN string) (string, error) {
	containerName := strings.Replace(APIServerFQDN, ".", "-", -1)

	return containerName, nil
}

func (exporter *AzureBlobExporter) GetNonKINDStorageContainerName(APIServerFQDN string) (string, error) {
	containerName := strings.Replace(APIServerFQDN, ".", "-", -1)

	//TODO I really dont like the line below, it makes for weird behaviour if e.g. .hcp. or -hcp- is in the fqdn for some reason other than being auto-added by AKS
	length := strings.Index(containerName, "-hcp-")

	if length == -1 {
		maxLength := len(containerName)
		length = int(math.Min(float64(maxLength), float64(maxContainerNameLength)))
	}

	containerName = containerName[:length]
	containerName = strings.TrimRight(containerName, "-")
	return containerName, nil
}

// Export implements the interface method
func (exporter *AzureBlobExporter) Export(files []string) error {
	APIServerFQDN, err := utils.GetAPIServerFQDN()
	if err != nil {
		return fmt.Errorf("Failed to get APIServerFQDN: %+v", err)
	}

	containerName, err := exporter.GetStorageContainerName(APIServerFQDN)
	if err != nil {
		return fmt.Errorf("Failed to get StorageContainerName: %+v", err)
	}

	ctx := context.Background()

	pipeline := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT_NAME")
	sasKey := os.Getenv("AZURE_BLOB_SAS_KEY")

	ses := utils.GetStorageEndpointSuffix()
	url, err := url.Parse(fmt.Sprintf("https://%s.blob.%s/%s%s", accountName, ses, containerName, sasKey))
	if err != nil {
		return fmt.Errorf("Failed to build blob container url: %+v", err)
	}

	containerURL := azblob.NewContainerURL(*url, pipeline)

	_, err = containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		storageError, ok := err.(azblob.StorageError)
		if ok {
			switch storageError.ServiceCode() {
			case azblob.ServiceCodeContainerAlreadyExists:
			default:
				return fmt.Errorf("Failed to create blob with storage error: %+v", err)
			}
		} else {
			return fmt.Errorf("Failed to create blob: %+v", err)
		}
	}

	for _, file := range files {
		appendBlobURL := containerURL.NewAppendBlobURL(strings.TrimPrefix(file, "/aks-periscope/"))

		blobGetPropertiesResponse, err := appendBlobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
		if err != nil {
			storageError, ok := err.(azblob.StorageError)
			if ok {
				switch storageError.ServiceCode() {
				case azblob.ServiceCodeBlobNotFound:
					_, err = appendBlobURL.Create(ctx, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})
					if err != nil {
						return fmt.Errorf("Fail to create blob for file %s: %+v", file, err)
					}
				default:
					return fmt.Errorf("Failed to create blob with storage error: %+v", err)
				}
			} else {
				return fmt.Errorf("Failed to create blob: %+v", err)
			}
		}

		var start int64
		if blobGetPropertiesResponse != nil {
			start = blobGetPropertiesResponse.ContentLength()
		}

		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("Fail to open file %s: %+v", file, err)
		}

		fileInfo, err := f.Stat()
		if err != nil {
			return fmt.Errorf("Fail to get file info for file %s: %+v", file, err)
		}

		end := fileInfo.Size()

		fileSize := end - start
		if fileSize > 0 {
			for start < end {
				lengthToWrite := end - start

				if lengthToWrite > azblob.AppendBlobMaxAppendBlockBytes {
					lengthToWrite = azblob.AppendBlobMaxAppendBlockBytes
				}

				b := make([]byte, lengthToWrite)
				_, err = f.ReadAt(b, start)
				if err != nil {
					return fmt.Errorf("Fail to read file %s: %+v", file, err)
				}

				log.Printf("\tappend blob file: %s, start position: %d, end position: %d\n", file, start, start+lengthToWrite)
				_, err = appendBlobURL.AppendBlock(ctx, bytes.NewReader(b), azblob.AppendBlobAccessConditions{}, nil)
				if err != nil {
					return fmt.Errorf("Fail to append file %s to blob: %+v", file, err)
				}

				start += lengthToWrite
			}
		}
	}

	return nil
}
