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

const (
	maxContainerNameLength = 63
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

	accountName := os.Getenv("AZURE_BLOB_ACCOUNT_NAME")
	sasKey := os.Getenv("AZURE_BLOB_SAS_KEY")

	containerName := strings.Replace(APIServerFQDN, ".", "-", -1)
	len := strings.Index(containerName, "-hcp-")
	if len == -1 {
		len = maxContainerNameLength
	}
	containerName = strings.TrimRight(containerName[:len], "-")

	ctx := context.Background()

	cred, err := azblob.NewSharedKeyCredential(accountName, sasKey)
	if err != nil {
		return fmt.Errorf("create SAS Key Credential: %w", err)
	}

	pipeline := azblob.NewPipeline(cred, azblob.PipelineOptions{})

	url, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))
	if err != nil {
		return fmt.Errorf("build blob container url: %w", err)
	}

	containerURL := azblob.NewContainerURL(*url, pipeline)

	if _, err = containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone); err != nil {
		storageError, ok := err.(azblob.StorageError)
		if ok {
			switch storageError.ServiceCode() {
			case azblob.ServiceCodeContainerAlreadyExists:
			default:
				return fmt.Errorf("create blob with storage error: %w", err)
			}
		} else {
			return fmt.Errorf("create blob: %w", err)
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
						return fmt.Errorf("create blob for file %s: %w", file, err)
					}
				default:
					return fmt.Errorf("create blob, storage error: %w", err)
				}
			} else {
				return fmt.Errorf("create blob, other error: %w", err)
			}
		}

		var start int64
		if blobGetPropertiesResponse != nil {
			start = blobGetPropertiesResponse.ContentLength()
		}

		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("open file %s: %w", file, err)
		}

		fileInfo, err := f.Stat()
		if err != nil {
			return fmt.Errorf("get file info for file %s: %w", file, err)
		}

		end := fileInfo.Size()

		for end-start > 0 {
			lengthToWrite := end - start

			if lengthToWrite > azblob.AppendBlobMaxAppendBlockBytes {
				lengthToWrite = azblob.AppendBlobMaxAppendBlockBytes
			}

			b := make([]byte, lengthToWrite)
			if _, err = f.ReadAt(b, start); err != nil {
				return fmt.Errorf("read file %s: %w", file, err)
			}

			log.Printf("\tappend blob file: %s, start position: %d, end position: %d", file, start, start+lengthToWrite)
			if _, err = appendBlobURL.AppendBlock(ctx, bytes.NewReader(b), azblob.AppendBlobAccessConditions{}, nil); err != nil {
				return fmt.Errorf("append file %s to blob: %w", file, err)
			}

			start += lengthToWrite
		}
	}

	return nil
}
