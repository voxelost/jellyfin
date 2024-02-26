package main

import (
	"context"
	"fmt"
	"io"
	"main/cache"
	"main/utils"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type Jellyfin struct {
	Service
	APIToken string
}

func NewJellyfin(cli *client.Client, cache *cache.RoseDBClient) (*Jellyfin, error) {
	port, err := nat.NewPort("tcp", "8096")
	if err != nil {
		return nil, err
	}

	hostPort, err := utils.GetFreePort()
	if err != nil {
		return nil, err
	}

	return &Jellyfin{
		Service: Service{
			ImageConfig: ImageConfig{
				RefString: "docker.io/jellyfin/jellyfin",
				Image: "jellyfin/jellyfin",
				ImagePullOptions: types.ImagePullOptions{},
			},
			ContainerConfig: ContainerConfig{
				PortMapping: nat.PortMap{
					port: []nat.PortBinding{
						{
							HostIP: "0.0.0.0",
							HostPort: fmt.Sprintf("%d", hostPort),
						},
					},
				},
				VolumeMapping: []mount.Mount{
					{
						Type: "bind",
						Source:         "/Users/voxelost/workspace/dev/jellyfin/.dev/volumes/jellyfin/cache",
						Target:         "/cache",
					},
					{
						Type: "bind",
						Source:         "/Users/voxelost/workspace/dev/jellyfin/.dev/volumes/jellyfin/config",
						Target:         "/config",
					},
					{
						Type: "bind",
						Source:         "/Users/voxelost/workspace/dev/jellyfin/.dev/volumes/jellyfin/media",
						Target:         "/media",
						ReadOnly: true,
					},
				},
			},
			dockerClient: cli,
			cacheClient: cache,
		},
	}, nil
}

func (j *Jellyfin) Start(ctx context.Context) error {
	err := j.EnsureImage(ctx)
	if err != nil {
		return err
	}

	err = j.EnsureContainer(ctx)
	if err != nil {
		return err
	}

	containerID, err := j.ID()
	if err != nil {
		return err
	}

	return j.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (j *Jellyfin) Logs(ctx context.Context) (io.ReadCloser, error) {
	containerId, err := j.ID()
	if err != nil {
		return nil, err
	}

	return j.dockerClient.ContainerLogs(ctx, containerId, container.LogsOptions{
		ShowStderr: true,
        ShowStdout: true,
        Timestamps: false,
        Follow:     true,
        Tail:       "40",
	})
}
