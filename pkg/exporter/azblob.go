package exporter

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
	"github.com/Azure/azure-storage-blob-go/azblob"
)

// ExportIntervalInSeconds defines interval for exporting data
var ExportIntervalInSeconds = 30

// ExportToAZBlob write data to azblob
func ExportToAZBlob() error {
	APIServerFQDN, err := utils.GetAPIServerFQDN()
	if err != nil {
		return err
	}
	containerName := strings.Replace(APIServerFQDN, ".", "-", -1)

	ctx := context.Background()

	p := azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{})
	accountName, sasKey := utils.GetAzureBlobCredential()

	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s%s", accountName, containerName, sasKey))
	containerURL := azblob.NewContainerURL(*URL, p)

	fmt.Printf("Creating a container named %s\n", containerName)
	_, err = containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
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

	ticker := time.NewTicker(time.Duration(ExportIntervalInSeconds) * time.Second)
	for {
		select {
		case <-ticker.C:
			err := exportData(ctx, containerURL)
			if err != nil {
				return err
			}
		}
	}
}

func exportData(ctx context.Context, containerURL azblob.ContainerURL) error {
	files, _ := listFilesInDir("/aks-diagnostic")
	for _, file := range files {
		blobURL := containerURL.NewBlockBlobURL(strings.Replace(file, "/aks-diagnostic/", "", -1))
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

func listFilesInDir(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
