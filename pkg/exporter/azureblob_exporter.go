package exporter

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azure/aks-periscope/pkg/blob"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/Azure/azure-storage-blob-go/azblob"
)

const (
	maxContainerNameLength = 63
)

// AzureBlobExporter defines an Azure Blob Exporter
type AzureBlobExporter struct {
	ctx          context.Context
	containerURL azblob.ContainerURL
}

var _ interfaces.Exporter = &AzureBlobExporter{}

// NewAzureBlobExporter initializes a new instance of AzureBlobExporter
func NewAzureBlobExporter() (*AzureBlobExporter, error) {
	sasKey := os.Getenv("AZURE_BLOB_SAS_KEY")
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT_NAME")
	containerName, err := getContainerName()
	if err != nil {
		return nil, err
	}

	var containerURL azblob.ContainerURL

	if sasKey != "" {
		containerURL, err = blob.CreateContainerURLFromSASKey(accountName, containerName, sasKey)
	} else {
		containerURL, err = blob.CreateContainerURLFromAssignedIdentity(accountName, containerName)
	}

	if err != nil {
		return nil, err
	}

	return &AzureBlobExporter{
		containerURL: containerURL,
		ctx:          context.Background(),
	}, nil
}

// Export implements the interface method
func (exporter *AzureBlobExporter) Export(files []string) error {
	_, err := exporter.containerURL.Create(exporter.ctx, azblob.Metadata{}, azblob.PublicAccessNone)
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

		appendBlobURL := exporter.containerURL.NewAppendBlobURL(strings.TrimPrefix(file, "/aks-periscope/"))

		blobGetPropertiesResponse, err := appendBlobURL.GetProperties(exporter.ctx, azblob.BlobAccessConditions{})
		if err != nil {
			storageError, ok := err.(azblob.StorageError)
			if ok {
				switch storageError.ServiceCode() {
				case azblob.ServiceCodeBlobNotFound:
					_, err = appendBlobURL.Create(exporter.ctx, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})
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
				_, err = appendBlobURL.AppendBlock(exporter.ctx, bytes.NewReader(b), azblob.AppendBlobAccessConditions{}, nil)
				if err != nil {
					return fmt.Errorf("Fail to append file %s to blob: %+v", file, err)
				}

				start += lengthToWrite
			}
		}
	}

	return nil
}

func getContainerName() (string, error) {
	apiServerFQDN, err := utils.GetAPIServerFQDN()
	if err != nil {
		return "", err
	}

	containerName := strings.Replace(apiServerFQDN, ".", "-", -1)
	len := strings.Index(containerName, "-hcp-")
	if len == -1 {
		len = maxContainerNameLength
	}

	return strings.TrimRight(containerName[:len], "-"), nil
}
