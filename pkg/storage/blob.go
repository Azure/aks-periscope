package storage

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
	"github.com/Azure/azure-storage-blob-go/azblob"
)

// WriteToBlob write data to blob
func WriteToBlob(cluster, containerName string, files []string) error {
	ctx := context.Background()

	p := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	accountName, sasKey := utils.GetAzureBlobCredential()

	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s%s", accountName, cluster, sasKey))
	containerURL := azblob.NewContainerURL(*URL, p)

	fmt.Printf("Creating a container named %s\n", containerName)
	_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		storageError, ok := err.(azblob.StorageError)
		if ok {
			switch storageError.ServiceCode() {
			case azblob.ServiceCodeContainerAlreadyExists:
			default:
				log.Fatal(err)
				return err
			}
		} else {
			log.Fatal(err)
			return err
		}
	}

	for _, file := range files {
		blobURL := containerURL.NewBlockBlobURL(strings.Replace(file, "/aks-diagnostic", containerName, -1))
		file, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
			return err
		}

		fmt.Printf("Uploading the file with blob name: %s\n", blobURL.String())
		_, err = azblob.UploadFileToBlockBlob(ctx, file, blobURL, azblob.UploadToBlockBlobOptions{
			BlockSize:   4 * 1024 * 1024,
			Parallelism: 16})
		if err != nil {
			log.Fatal(err)
			return err
		}
	}

	return nil
}
