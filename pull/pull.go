package pull

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
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

func (c *Client) FetchLayersFromManifest(tag string) error {
	if len(c.manifest.Layers) == 0 {
		return fmt.Errorf("no layers in manifest: %q", c.manifest)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	layerDir := filepath.Join(home, ".local", "share", "capsule", "layers", c.repo)
	for _, layer := range c.manifest.Layers {
		if err := func() error {
			u, err := url.Parse(registry)
			if err != nil {
				return err
			}
			u.Path = path.Join(u.Path, c.repo, "blobs", layer.Digest)
			uri := u.String()
			req, err := c.authorizedRequest(http.MethodGet, uri)
			if err != nil {
				return err
			}
			resp, err := c.httpClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("wanted 200 OK but got: %s", resp.Status)
			}

			if err := os.MkdirAll(layerDir, 0o755); err != nil {
				return err
			}
			f, err := os.Create(filepath.Join(layerDir, layer.Digest))
			if err != nil {
				return err
			}
			defer f.Close()

			// todo download to a temp file and rename on success to avoid partial blobs
			// todo verify the downloaded content matches layer.Digest
			_, err = io.Copy(f, resp.Body)
			return err
		}(); err != nil {
			return err
		}
	}
	return nil
}

type layerReadCloser struct {
	file *os.File
	gzip *gzip.Reader
}

func (l *layerReadCloser) Close() error {
	var err error
	if l.gzip != nil {
		err = l.gzip.Close()
	}
	if closeErr := l.file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func openLayerTarReader(p, mediaType string) (*tar.Reader, io.Closer, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, nil, err
	}

	closer := &layerReadCloser{file: f}
	var r io.Reader = f

	switch mediaType {
	case "application/vnd.oci.image.layer.v1.tar+gzip",
		"application/vnd.docker.image.rootfs.diff.tar.gzip":
		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, nil, err
		}
		closer.gzip = gz
		r = gz
	}

	return tar.NewReader(r), closer, nil
}

func removeDirContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		p := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

func resolveRootfsPath(root string, tarPath string) (string, error) {
	clean := path.Clean("/" + tarPath)
	rel := strings.TrimPrefix(clean, "/")
	if rel == "." {
		rel = ""
	}

	full := filepath.Join(root, filepath.FromSlash(rel))
	relToRoot, err := filepath.Rel(root, full)
	if err != nil {
		return "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("tar path escapes rootfs: %q", tarPath)
	}
	return full, nil
}

func (c *Client) ApplyLayers(tag string) error {
	if len(c.manifest.Layers) == 0 {
		return fmt.Errorf("no layers in manifest: %q", c.manifest)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	layerDir := filepath.Join(home, ".local", "share", "capsule", "layers", c.repo)
	rootfsDir := filepath.Join(home, ".local", "share", "capsule", "rootfs", c.repo, tag)

	for _, layer := range c.manifest.Layers {
		if err := func() error {

			p1 := filepath.Join(layerDir, layer.Digest)
			t1, closer, err := openLayerTarReader(p1, layer.MediaType)
			if err != nil {
				return err
			}
			defer closer.Close()

			for {
				hdr, err := t1.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				dir := path.Dir(hdr.Name)
				base := path.Base(hdr.Name)
				if base == ".wh..wh..opq" {
					wo, err := resolveRootfsPath(rootfsDir, dir)
					if err != nil {
						return err
					}
					if err := removeDirContents(wo); err != nil {
						return err
					}
				} else if strings.HasPrefix(base, ".wh.") {
					name := strings.TrimPrefix(base, ".wh.")
					wo, err := resolveRootfsPath(rootfsDir, path.Join(dir, name))
					if err != nil {
						return err
					}
					if err := os.RemoveAll(wo); err != nil {
						return err
					}
				} else {
					continue
				}
			}

			p2 := filepath.Join(layerDir, layer.Digest)
			t2, closer2, err := openLayerTarReader(p2, layer.MediaType)
			if err != nil {
				return err
			}
			defer closer2.Close()

			for {
				hdr, err := t2.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				base := path.Base(hdr.Name)
				if base != ".wh..wh..opq" && !strings.HasPrefix(base, ".wh.") {
					// todo extract archieve
				}
			}
			return nil
		}(); err != nil {
			return err
		}
	}
	return nil
}

// pulling a layer
// GET /v2/<name>/blobs/<digest>

// pulling a manifest
// GET /v2/<name>/manifests/<reference>
