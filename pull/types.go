package pull

// Parameters
// ?service=registry.docker.io
// scope=repository:library/alpine:pull

type authTokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

type manifestList struct {
	SchemaVersion int                `json:"schemaVersion"`
	MediaType     string             `json:"mediaType"`
	Manifests     []manifestListItem `json:"manifests"`
}

type manifestListItem struct {
	MediaType string               `json:"mediaType"`
	Digest    string               `json:"digest"`
	Size      int64                `json:"size"`
	Platform  manifestListPlatform `json:"platform"`
}

type manifestListPlatform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Variant      string `json:"variant,omitempty"`
}

type manifestResponse struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	Config        manifestConfig  `json:"config"`
	Layers        []manifestLayer `json:"layers"`
}

type manifestConfig struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

type manifestLayer struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}
