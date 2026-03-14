package pull

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"time"
)

const registry string = "https://registry-1.docker.io/v2/"
const auth_url = "https://auth.docker.io/token"
const authRefreshBuffer = 15 * time.Second

type Client struct {
	httpClient *http.Client
	repo       string
	token      authTokenResponse
	manifest   manifestResponse
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
	if !c.IsAuthenticated() {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token.Token)
	return req, nil
}

func (c *Client) GetManifest(tag string) error {
	u, err := url.Parse(registry)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, c.repo, "manifests", tag)
	uri := u.String()

	req, err := c.authorizedRequest(http.MethodGet, uri)
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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wanted status 200 OK but got: %s", resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	_ = contentType

	arch := runtime.GOARCH
	sys := runtime.GOOS

	var ml manifestList = manifestList{}
	var manifest manifestResponse = manifestResponse{}

	if contentType == "application/vnd.oci.image.index.v1+json" ||
		contentType == "application/vnd.docker.distribution.manifest.list.v2+json" {
		if err := json.NewDecoder(resp.Body).Decode(&ml); err != nil {
			return err
		}

		for _, manifestItem := range ml.Manifests {
			if manifestItem.Platform.OS == sys && manifestItem.Platform.Architecture == arch {
				u, err := url.Parse(registry)
				if err != nil {
					return err
				}
				u.Path = path.Join(u.Path, c.repo, "manifests", manifestItem.Digest)
				uri := u.String()
				req, err := c.authorizedRequest(http.MethodGet, uri)
				if err != nil {
					return err
				}
				req.Header.Set(
					"Accept",
					"application/vnd.oci.image.manifest.v1+json, "+
						"application/vnd.docker.distribution.manifest.v2+json",
				)
				resp, err = c.httpClient.Do(req)
				if err != nil {
					return err
				}
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("wanted status 200 OK but got: %s", resp.Status)
				}
				contentType = resp.Header.Get("Content-Type")
				if contentType != "application/vnd.oci.image.manifest.v1+json" &&
					contentType != "application/vnd.docker.distribution.manifest.v2+json" {
					return fmt.Errorf("expected manifest content type but got: %s", contentType)
				}
				defer resp.Body.Close()
				if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
					return err
				}
				c.manifest = manifest
				return nil
			}
		}
	}

	if contentType == "application/vnd.oci.image.manifest.v1+json" ||
		contentType == "application/vnd.docker.distribution.manifest.v2+json" {
		if err = json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
			return err
		}
		c.manifest = manifest
		return nil
	}

	return fmt.Errorf("could not find manifest for %s:%s", c.repo, tag)
}

func (c *Client) IsAuthenticated() bool {
	if c.token.Token == "" {
		return false
	}

	if c.token.ExpiresIn <= 0 || c.token.IssuedAt == "" {
		return false
	}

	issuedAt, err := time.Parse(time.RFC3339, c.token.IssuedAt)
	if err != nil {
		return false
	}

	expiresAt := issuedAt.Add(time.Duration(c.token.ExpiresIn) * time.Second)
	return time.Now().Add(authRefreshBuffer).Before(expiresAt)
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
