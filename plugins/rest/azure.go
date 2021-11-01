package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	azureIMDSEndpoint = "http://169.254.169.254/metadata/identity/oauth2/token"
	timeout = time.Duration(5) * time.Second
)

// azureManagedIdentitiesToken holds a token for managed identities for Azure resources
type azureManagedIdentitiesToken struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	ExpiresOn    string `json:"expires_on"`
	NotBefore    string `json:"not_before"`
	Resource     string `json:"resource"`
	TokenType    string `json:"token_type"`
}

// azureManagedIdentitiesError represents an error fetching an azureManagedIdentitiesToken
type azureManagedIdentitiesError struct {
	err         string `json:"error"`
	description string `json:"error_description"`
	endpoint    string
	statusCode  int
}

func (e *azureManagedIdentitiesError) Error() string {
	return fmt.Sprintf("%v %s retrieving azure token from %s: %s", e.statusCode, e.err, e.endpoint, e.description)
}

// azureManagedIdentitiesAuthPlugin uses an azureManagedIdentitiesToken.AccessToken for bearer authorization
type azureManagedIdentitiesAuthPlugin struct {
	Endpoint   string `json:"endpoint"`
	APIVersion string `json:"api_version"`
	Resource   string `json:"resource"`
	ObjectId   string `json:"object_id"`
	ClientId   string `json:"client_id"`
	MiResId    string `json:"mi_res_id"`
}

func (ap *azureManagedIdentitiesAuthPlugin) NewClient(c Config) (*http.Client, error) {
	if ap.APIVersion == "" {
		return nil, errors.New("API version is required when the azure managed identities plugin is enabled")
	}

	if ap.Resource == "" {
		return nil, errors.New("resource URI is required when the azure managed identities plugin is enabled")
	}

	if ap.Endpoint == "" {
		ap.Endpoint = azureIMDSEndpoint
	}

	t, err := DefaultTLSConfig(c)
	if err != nil {
		return nil, err
	}

	return DefaultRoundTripperClient(t, *c.ResponseHeaderTimeoutSeconds), nil
}

func (ap *azureManagedIdentitiesAuthPlugin) Prepare(req *http.Request) error {
	token, err := azureManagedIdentitiesTokenRequest(
		ap.Endpoint, ap.APIVersion, ap.Resource,
		ap.ObjectId, ap.ClientId, ap.MiResId,
	)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token.AccessToken))
	return nil
}

// azureManagedIdentitiesTokenRequest fetches an azureManagedIdentitiesToken
func azureManagedIdentitiesTokenRequest(
	endpoint, apiVersion, resource, objectId, clientId, miResId string,
) (azureManagedIdentitiesToken, error) {
	e := fmt.Sprintf("%s?api-version=%s&resource=%s", endpoint, apiVersion, resource)

	if objectId != "" {
		e += fmt.Sprintf("&object_id=%s", objectId)
	}

	if clientId != "" {
		e += fmt.Sprintf("&client_id=%s", clientId)
	}

	if miResId != "" {
		e += fmt.Sprintf("&mi_res_id=%s", miResId)
	}

	request, err := http.NewRequest("GET", e, nil)
	if err != nil {
		return azureManagedIdentitiesToken{}, err
	}
	request.Header.Add("Metadata", "true")

	httpClient := http.Client{Timeout: timeout}
	response, err := httpClient.Do(request)
	if err != nil {
		return azureManagedIdentitiesToken{}, err
	}

	data, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return azureManagedIdentitiesToken{}, err
	}

	if s := response.StatusCode; s != http.StatusOK {
		var azureError azureManagedIdentitiesError
		err = json.Unmarshal(data, &azureError)
		if err != nil {
			return azureManagedIdentitiesToken{}, err
		}

		azureError.endpoint = e
		azureError.statusCode = s
		return azureManagedIdentitiesToken{}, &azureError
	}

	var accessToken azureManagedIdentitiesToken
	err = json.Unmarshal(data, &accessToken)
	if err != nil {
		return azureManagedIdentitiesToken{}, err
	}

	return accessToken, nil
}
