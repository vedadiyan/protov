package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const REPO = "https://api.github.com/repos/%s/%s/releases/latest"

type assets []asset
type release string

type githubReposResponse struct {
	Assets assets `json:"assets"`
}

type asset struct {
	BrowserDownloadURL release `json:"browser_download_url"`
}

func (a assets) GetReleaseFor(os OS, arch ARCH) (release, error) {
	osAndArch, err := combineOSandArch(os, arch)
	if err != nil {
		return "", err
	}
	for _, i := range a {
		if strings.Contains(string(i.BrowserDownloadURL), osAndArch) {
			return i.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("not found")
}

func (r release) Download() (int64, io.ReadCloser, error) {
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

func LatestTag(owner string, repo string) (*githubReposResponse, error) {
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

	var repoResponse githubReposResponse
	if err := json.Unmarshal(bytes, &repoResponse); err != nil {
		return nil, err
	}

	return &repoResponse, nil
}

func combineOSandArch(os OS, arch ARCH) (string, error) {
	switch os {
	case OS_WINDOWS:
		{
			switch arch {
			case ARCH_AMD64:
				{
					return "win64", nil
				}
			}
		}
	case OS_DARWIN:
		{
			switch arch {
			case ARCH_AMD64:
				{
					return "osx-x86_64", nil
				}
			case ARCH_ARM64:
				{
					return "osx-aarch_64", nil
				}
			}
		}
	case OS_LINUX:
		{
			switch arch {
			case ARCH_AMD64:
				{
					return "linux-x86_32", nil
				}
			case ARCH_ARM64:
				{
					return "linux-aarch_64", nil
				}
			}
		}
	}
	return "", fmt.Errorf("%s-%s is not supported", os, arch)
}
