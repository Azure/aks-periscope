package storage

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	blob "github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// WriteToBlob write data to blob
func WriteToBlob(containerName string, files []string) error {
	ctx := context.Background()

	accountName, accountKey := utils.GetAzureBlobLogin()
	credential, err := blob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		log.Fatal(err)
		return err
	}
	p := blob.NewPipeline(credential, blob.PipelineOptions{})

	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))
	containerURL := blob.NewContainerURL(*URL, p)

	fmt.Printf("Creating a container named %s\n", containerName)
	_, err = containerURL.Create(ctx, blob.Metadata{}, blob.PublicAccessNone)
	if err != nil {
		storageError, ok := err.(blob.StorageError)
		if ok {
			switch storageError.ServiceCode() {
			case blob.ServiceCodeContainerAlreadyExists:
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
		blobURL := containerURL.NewBlockBlobURL(strings.TrimPrefix(file, "/aks-diagnostic/"))
		file, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
			return err
		}

		fmt.Printf("Uploading the file with blob name: %s\n", blobURL.String())
		_, err = blob.UploadFileToBlockBlob(ctx, file, blobURL, blob.UploadToBlockBlobOptions{
			BlockSize:   4 * 1024 * 1024,
			Parallelism: 16})
		if err != nil {
			log.Fatal(err)
			return err
		}
	}

	return nil
}
