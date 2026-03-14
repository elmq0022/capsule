package pull

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// const registry string = "https://registry-1.docker.io/v2/library"
const auth_url = "https://auth.docker.io/token"

// Parameters
// ?service=registry.docker.io
// scope=repository:library/alpine:pull

type authTokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

type Client struct {
	httpClient *http.Client
	repo       string
	token      authTokenResponse
}

func NewClient(repo string) *Client {
	return &Client{
		repo:       repo,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// todo auth
func (c *Client) Authenticate() error {
	req, err := http.NewRequest(http.MethodGet, auth_url, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Set("service", "registry.docker.io")
	q.Set("scope", fmt.Sprintf("repository:%s:pull", c.repo))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200 status but got: %s", resp.Status)
	}

	jwt := authTokenResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&jwt); err != nil {
		return fmt.Errorf("could not decode response body: %w", err)
	}
	c.token = jwt

	return nil
}

// todo pull manifests

// todo parse layers from manifest

// todo fetch each layer

// todo create the image dir if it doesn't exist
// and delete the contents

// todo unzip the layers to the rootfs
