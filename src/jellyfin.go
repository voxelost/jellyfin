package main

import (
	"context"
	"main/cache"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type Jellyfin struct {
	Service
	APIToken string
}

func NewJellyfin(ctx context.Context, cli *client.Client, cache *cache.RoseDBClient) (*Jellyfin, error) {
	service, err := NewService(
		ctx,
		ImageConfig{
			RefString:        "docker.io/jellyfin/jellyfin",
			Image:            "jellyfin/jellyfin",
			ImagePullOptions: types.ImagePullOptions{},
		},
		ContainerConfig{
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
		cli, cache, []int{8096},
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
