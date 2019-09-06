package exporter

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
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

	ctx := context.Background()

	p := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT_NAME")
	sasKey := os.Getenv("AZURE_BLOB_SAS_KEY")

	URL, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s%s", accountName, containerName, sasKey))
	if err != nil {
		return fmt.Errorf("Fail to build blob container url: %+v", err)
	}

	containerURL := azblob.NewContainerURL(*URL, p)

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
		blobURL := containerURL.NewBlockBlobURL(strings.Replace(file, "/aks-diagnostic/", "", -1))
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("Fail to open file %s: %+v", file, err)
		}

		_, err = azblob.UploadFileToBlockBlob(ctx, f, blobURL, azblob.UploadToBlockBlobOptions{
			BlockSize:   4 * 1024 * 1024,
			Parallelism: 16})
		if err != nil {
			return fmt.Errorf("Failed to upload file %s to blob: %+v", file, err)
		}
	}

	return nil
}
