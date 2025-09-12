package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const REPO = "https://api.github.com/repos/%s/%s/releases/latest"

type Assets []Asset
type Release string

type GithubReposResponse struct {
	Assets Assets `json:"assets"`
}

type Asset struct {
	BrowserDownloadURL Release `json:"browser_download_url"`
}

func (r Release) Download() (int64, io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, string(r), http.NoBody)
	if err != nil {
		return 0, nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	if res.StatusCode/100 != 2 {
		return 0, nil, fmt.Errorf("download failed with status code %d", res.StatusCode)
	}
	return res.ContentLength, res.Body, nil
}

func LatestTag(owner string, repo string) (*GithubReposResponse, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*30)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(REPO, owner, repo), http.NoBody)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var repoResponse GithubReposResponse
	if err := json.Unmarshal(bytes, &repoResponse); err != nil {
		return nil, err
	}

	return &repoResponse, nil
}
