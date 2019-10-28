package exporter

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/Azure/azure-storage-blob-go/azblob"
)

// AzureBlobExporter defines an Azure Blob Exporter
type AzureBlobExporter struct{}

var _ interfaces.Exporter = &AzureBlobExporter{}

// Export implements the interface method
func (exporter *AzureBlobExporter) Export(files []string) error {
	APIServerFQDN, err := utils.GetAPIServerFQDN()
	if err != nil {
		return err
	}

	containerName := strings.Replace(APIServerFQDN, ".", "-", -1)
	len := strings.Index(containerName, "-hcp-")
	containerName = containerName[:len]

	ctx := context.Background()

	pipeline := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT_NAME")
	sasKey := os.Getenv("AZURE_BLOB_SAS_KEY")

	url, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s%s", accountName, containerName, sasKey))
	if err != nil {
		return fmt.Errorf("Fail to build blob container url: %+v", err)
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
