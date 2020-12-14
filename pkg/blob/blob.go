package blob

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/Azure/aks-periscope/pkg/authentication"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/adal"
)

const (
	tokenRefreshTolerance = 300
)

// CreateContainerURLFromSASKey creates a ContainerURL from a SAS Key
func CreateContainerURLFromSASKey(accountName string, containerName string, sasKey string) (azblob.ContainerURL, error) {
	credentials := azblob.NewAnonymousCredential()

	return createContainerURL(credentials, accountName, containerName, sasKey)
}

// CreateContainerURLFromAssignedIdentity creates a ContainerURL from the assigned identity
func CreateContainerURLFromAssignedIdentity(accountName string, containerName string) (azblob.ContainerURL, error) {
	// The value can be found at azure.PublicCloud.ResourceIdentifiers.Storage
	spToken, err := authentication.GetTokenFromAssignedID("https://storage.azure.com/")
	if err != nil {
		return azblob.ContainerURL{}, err
	}

	err = spToken.Refresh()
	if err != nil {
		return azblob.ContainerURL{}, fmt.Errorf("get token: %+v", err)
	}

	token := spToken.Token()
	if token.IsZero() {
		return azblob.ContainerURL{}, fmt.Errorf("cannot acquire initial token from the SP Token: %+v", token)
	}

	credentials := azblob.NewTokenCredential(token.AccessToken, defaultTokenRefreshFunction(spToken))
	return createContainerURL(credentials, accountName, containerName, "")
}

func createContainerURL(credentials azblob.Credential, accountName string, containerName string, sasKey string) (azblob.ContainerURL, error) {
	pipeline := azblob.NewPipeline(credentials, azblob.PipelineOptions{})

	url, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s%s", accountName, containerName, sasKey))
	if err != nil {
		return azblob.ContainerURL{}, fmt.Errorf("build blob container url: %+v", err)
	}

	containerURL := azblob.NewContainerURL(*url, pipeline)

	return containerURL, nil
}

var defaultTokenRefreshFunction = func(spToken *adal.ServicePrincipalToken) func(credential azblob.TokenCredential) time.Duration {
	return func(credential azblob.TokenCredential) time.Duration {
		if err := spToken.Refresh(); err != nil {
			fmt.Printf("refresh token: %+v", err)
			return 0
		}

		expiresIn, err := strconv.ParseInt(string(spToken.Token().ExpiresIn), 10, 64)
		if err != nil {
			fmt.Printf("new token expiresIn cannot be parsed: %+v", err)
			return 0
		}

		credential.SetToken(spToken.Token().AccessToken)

		return time.Duration(expiresIn-tokenRefreshTolerance) * time.Second
	}
}
