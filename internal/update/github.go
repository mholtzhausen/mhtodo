package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Release is a GitHub release we can install from.
type Release struct {
	TagName   string // e.g. "v1.2.0"
	Version   string // tag without leading v
	AssetName string
	AssetURL  string // browser_download_url
	HTMLURL   string
	Digest    string // "sha256:<hex>" when GitHub provides it
}

// GitHubClient fetches release metadata and assets.
type GitHubClient struct {
	HTTP      *http.Client
	OwnerRepo string // default OwnerRepo
	Token     string // optional Bearer
	APIBase   string // default https://api.github.com
}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		HTTP:      &http.Client{Timeout: 60 * time.Second},
		OwnerRepo: OwnerRepo,
		Token:     githubToken(),
		APIBase:   "https://api.github.com",
	}
}

func githubToken() string {
	for _, k := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

// LatestRelease returns the newest published release that has a matching
// linux tarball for arch (amd64|arm64).
func (c *GitHubClient) LatestRelease(arch string) (Release, error) {
	if c.OwnerRepo == "" {
		c.OwnerRepo = OwnerRepo
	}
	if c.APIBase == "" {
		c.APIBase = "https://api.github.com"
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 60 * time.Second}
	}
	url := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimRight(c.APIBase, "/"), c.OwnerRepo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mhtodo-update")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Release{}, fmt.Errorf("read github response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("github api: %s: %s", resp.Status, truncate(string(body), 200))
	}
	var raw ghRelease
	if err := json.Unmarshal(body, &raw); err != nil {
		return Release{}, fmt.Errorf("decode github release: %w", err)
	}
	ver := NormalizeVersion(raw.TagName)
	if ver == "" {
		return Release{}, fmt.Errorf("github release has empty tag_name")
	}
	want := AssetName(ver, arch)
	for _, a := range raw.Assets {
		if a.Name == want && a.BrowserDownloadURL != "" {
			return Release{
				TagName:   raw.TagName,
				Version:   ver,
				AssetName: a.Name,
				AssetURL:  a.BrowserDownloadURL,
				HTMLURL:   raw.HTMLURL,
				Digest:    a.Digest,
			}, nil
		}
	}
	return Release{}, fmt.Errorf("release v%s has no asset %s", ver, want)
}

// Download writes the release asset to destPath.
func (c *GitHubClient) Download(assetURL, destPath string) error {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 5 * time.Minute}
	}
	req, err := http.NewRequest(http.MethodGet, assetURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "mhtodo-update")
	req.Header.Set("Accept", "application/octet-stream")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("download: %s: %s", resp.Status, truncate(string(body), 200))
	}
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write download: %w", err)
	}
	return f.Close()
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
