package pull

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

const registry string = "https://registry-1.docker.io/v2/"
const auth_url = "https://auth.docker.io/token"

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

func (c *Client) authorizedRequest(method, url string) (*http.Request, error) {
	if c.token.Token == "" {
		return nil, fmt.Errorf("authorized requests need a token")
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token.Token)
	return req, nil
}

func (c *Client) GetManifest(tag string) error {
	err := c.Authenticate()
	if err != nil {
		return err
	}

	u, err := url.Parse(registry)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, c.repo, "manifests", tag)
	url := u.String()

	req, err := c.authorizedRequest(http.MethodGet, url)
	if err != nil {
		return err
	}

	req.Header.Set(
		"Accept",
		"application/vnd.oci.image.index.v1+json, "+
			"application/vnd.docker.distribution.manifest.list.v2+json, "+
			"application/vnd.oci.image.manifest.v1+json, "+
			"application/vnd.docker.distribution.manifest.v2+json",
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	_ = contentType

	return nil
}

// todo parse layers from manifest

// todo fetch each layer

// todo create the image dir if it doesn't exist
// and delete the contents

// todo unzip the layers to the rootfs

// pulling a layer
// GET /v2/<name>/blobs/<digest>

// pulling a manifest
// GET /v2/<name>/manifests/<reference>
