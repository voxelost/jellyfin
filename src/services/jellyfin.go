package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"main/services/service"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type Jellyfin struct {
	service.Service
	apiToken *string
}

func NewJellyfin(ctx context.Context, cli *client.Client) (*Jellyfin, error) {
	service, err := service.NewService(
		ctx,
		service.ImageConfig{
			RefString:        "docker.io/jellyfin/jellyfin:2024030405",
			Image:            "jellyfin/jellyfin:2024030405",
			ImagePullOptions: types.ImagePullOptions{},
		},
		service.ContainerConfig{
			VolumeMapping: []mount.Mount{
				{
					Type:   "volume",
					Source: "jellyfin-config",
					Target: "/config",
				},
			},
			Env: []string{
				"PUID=1000",
				"PGID=1000",
				"TZ=Europe/Warsaw",
			},
		},
		cli, []int{8096},
	)

	if err != nil {
		return nil, err
	}

	return &Jellyfin{
		Service: *service,
	}, nil
}

func (j *Jellyfin) BaseSetup() error {
	// setup language
	_, err := j.HttpPost("/Startup/Configuration", `{"UICulture":"en-US","MetadataCountryCode":"US","PreferredMetadataLanguage":"en"}`)
	if err != nil {
		return err
	}

	_, err = j.HttpGet("/Startup/User")
	if err != nil {
		return err
	}

	// setup language
	_, err = j.HttpPost("/Startup/User", `{"Name":"root","Password":"root"}`)
	if err != nil {
		return err
	}

	// setup remote access
	_, err = j.HttpPost("/Startup/RemoteAccess", `{"EnableRemoteAccess":true,"EnableAutomaticPortMapping":false}`)
	if err != nil {
		return err
	}

	// finish setup
	_, err = j.HttpPost("/Startup/Complete", ``)
	return err
}

func (j *Jellyfin) ApiToken() (string, error) {
	if j.apiToken != nil {
		return *j.apiToken, nil
	}

	resp, err := j.HttpPost("/Users/authenticatebyname", `{"Username":"root","Pw":"root"}`)
	if err != nil {
		return "", err
	}

	type tokenWrapper struct {
		Token string `json:"AccessToken"`
	}

	tw := tokenWrapper{}

	bufsize := 4096
	if resp.ContentLength > 0 {
		bufsize = int(resp.ContentLength)
	}
	buf := make([]byte, bufsize)
	defer resp.Body.Close()

	resp.Body.Read(buf)
	err = json.Unmarshal(buf[:len(bytes.Trim(buf, "\x00"))], &tw)
	if err != nil {
		return "", err
	}

	slog.Debug(fmt.Sprintf("got token: %s", tw.Token))
	j.apiToken = &tw.Token

	return j.ApiToken()
}

func (j *Jellyfin) HttpGet(path string) (*http.Response, error) {
	apiURL, err := j.ApiAddress()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, apiURL.JoinPath(path).String(), strings.NewReader(""))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", `MediaBrowser Client="Jellyfin Web", Device="Firefox", DeviceId="TW96aWxsYS81LjAgKE1hY2ludG9zaDsgSW50ZWwgTWFjIE9TIFggMTAuMTU7IHJ2OjEyMy4wKSBHZWNrby8yMDEwMDEwMSBGaXJlZm94LzEyMy4wfDE3MDk1ODgyMDI2OTM1", Version="10.9.0"`)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBodyReader := resp.Body
		defer respBodyReader.Close()

		var buf []byte
		if resp.ContentLength > 0 {
			buf = make([]byte, resp.ContentLength)
			respBodyReader.Read(buf)
		} else {
			buf = []byte(resp.Status)
		}
		return nil, fmt.Errorf("response status code is not 2XX: %s", buf)
	}

	return resp, err
}

func (j *Jellyfin) HttpPost(path string, body string) (*http.Response, error) {
	apiURL, err := j.ApiAddress()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, apiURL.JoinPath(path).String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", `MediaBrowser Client="Jellyfin Web", Device="Firefox", DeviceId="TW96aWxsYS81LjAgKE1hY2ludG9zaDsgSW50ZWwgTWFjIE9TIFggMTAuMTU7IHJ2OjEyMy4wKSBHZWNrby8yMDEwMDEwMSBGaXJlZm94LzEyMy4wfDE3MDk1ODgyMDI2OTM1", Version="10.9.0"`)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBodyReader := resp.Body
		defer respBodyReader.Close()

		var buf []byte
		if resp.ContentLength > 0 {
			buf = make([]byte, resp.ContentLength)
			respBodyReader.Read(buf)
		} else {
			buf = []byte(resp.Status)
		}

		return nil, fmt.Errorf("response status code is not 2XX: %s", buf)
	}

	return resp, err
}
