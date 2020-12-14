package authentication

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
)

// GetTokenFromAssignedID retrieves a token from the environment config
func GetTokenFromAssignedID(resource string) (*adal.ServicePrincipalToken, error) {
	msiEndpoint, err := adal.GetMSIEndpoint()
	if err != nil {
		return nil, fmt.Errorf("get MSI endpoint: %+v", err)
	}

	spToken, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
	if err != nil {
		return nil, fmt.Errorf("acquire a SP token using the MSI endpoint (%s): %+v", msiEndpoint, err)
	}

	return spToken, nil
}
