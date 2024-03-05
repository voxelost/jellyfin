package services

import (
	"context"
	"main/services/service"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type Jellyseerr struct {
	service.Service

	apiToken *string
}

func NewJellyseerr(ctx context.Context, cli *client.Client) (*Jellyseerr, error) {
	service, err := service.NewService(
		ctx,
		service.ImageConfig{
			RefString:        "docker.io/fallenbagel/jellyseerr:1.7.0",
			Image:            "fallenbagel/jellyseerr:1.7.0",
			ImagePullOptions: types.ImagePullOptions{},
		},
		service.ContainerConfig{
			VolumeMapping: []mount.Mount{
				{
					Type:   "volume",
					Source: "jellyseerr-config",
					Target: "/app/config",
				},
			},
			Env: []string{
				// "PUID=1000",
				// "PGID=1000",
				"TZ=Europe/Warsaw",
				"LOG_LEVEL=debug",
			},
		},
		cli, []int{5055},
	)
	if err != nil {
		return nil, err
	}

	return &Jellyseerr{
		Service: *service,
	}, nil
}
