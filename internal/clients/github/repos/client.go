package githubrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vedadiyan/protov/internal/common"
)

const REPO = "https://api.github.com/repos/protocolbuffers/protobuf/releases/latest"

type Assets []Asset
type Release string

type GithubReposResponse struct {
	URL             string    `json:"url"`
	AssetsURL       string    `json:"assets_url"`
	UploadURL       string    `json:"upload_url"`
	HTMLURL         string    `json:"html_url"`
	ID              int64     `json:"id"`
	Author          Author    `json:"author"`
	NodeID          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Immutable       bool      `json:"immutable"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
	PublishedAt     string    `json:"published_at"`
	Assets          Assets    `json:"assets"`
	TarballURL      string    `json:"tarball_url"`
	ZipballURL      string    `json:"zipball_url"`
	Body            string    `json:"body"`
	Reactions       Reactions `json:"reactions"`
}

type Asset struct {
	URL                string  `json:"url"`
	ID                 int64   `json:"id"`
	NodeID             string  `json:"node_id"`
	Name               string  `json:"name"`
	Label              string  `json:"label"`
	Uploader           Author  `json:"uploader"`
	ContentType        string  `json:"content_type"`
	State              string  `json:"state"`
	Size               int64   `json:"size"`
	Digest             string  `json:"digest"`
	DownloadCount      int64   `json:"download_count"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
	BrowserDownloadURL Release `json:"browser_download_url"`
}

type Author struct {
	Login             string `json:"login"`
	ID                int64  `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	UserViewType      string `json:"user_view_type"`
	SiteAdmin         bool   `json:"site_admin"`
}

type Reactions struct {
	URL        string `json:"url"`
	TotalCount int64  `json:"total_count"`
	The1       int64  `json:"+1"`
	Reactions1 int64  `json:"-1"`
	Laugh      int64  `json:"laugh"`
	Hooray     int64  `json:"hooray"`
	Confused   int64  `json:"confused"`
	Heart      int64  `json:"heart"`
	Rocket     int64  `json:"rocket"`
	Eyes       int64  `json:"eyes"`
}

func (a Assets) GetReleaseFor(os common.OS, arch common.ARCH) (Release, error) {
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

func LatestTag() (*GithubReposResponse, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*30)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, REPO, http.NoBody)
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

func combineOSandArch(os common.OS, arch common.ARCH) (string, error) {
	switch os {
	case common.OS_WINDOWS:
		{
			switch arch {
			case common.ARCH_AMD64:
				{
					return "win64", nil
				}
			}
		}
	case common.OS_DARWIN:
		{
			switch arch {
			case common.ARCH_AMD64:
				{
					return "osx-x86_64", nil
				}
			case common.ARCH_ARM64:
				{
					return "osx-aarch_64", nil
				}
			}
		}
	case common.OS_LINUX:
		{
			switch arch {
			case common.ARCH_AMD64:
				{
					return "linux-x86_32", nil
				}
			case common.ARCH_ARM64:
				{
					return "linux-aarch_64", nil
				}
			}
		}
	}
	return "", fmt.Errorf("%s-%s is not supported", os, arch)
}
