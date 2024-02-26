package main

import (
	"context"
	"fmt"
	"io"
	"main/cache"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type ImageConfig struct {
	RefString string
	Image string
	ImagePullOptions types.ImagePullOptions
}

type ContainerConfig struct {
	PortMapping nat.PortMap
	VolumeMapping []mount.Mount
}

type Service struct {
	ImageConfig ImageConfig
	ContainerConfig ContainerConfig

	dockerClient *client.Client
	cacheClient *cache.RoseDBClient
	containerID *string
}

func (s *Service) ID() (string, error) {
	if s.containerID == nil {
		return "", fmt.Errorf("service not created yet")
	}

	return *s.containerID, nil
}

func (s *Service) EnsureImage(ctx context.Context) error {
	reader, err := s.dockerClient.ImagePull(ctx, s.ImageConfig.RefString, s.ImageConfig.ImagePullOptions)
	if err != nil {
		return err
	}

	defer reader.Close()
	io.Copy(io.Discard, reader)

	return nil
}

func (s *Service) EnsureContainer(ctx context.Context) (error) {
	// todo: try to get container id from cache
	volumeMap := make(map[string]struct{})
	for _, volume := range s.ContainerConfig.VolumeMapping {
		volumeMap[volume.Target] = struct{}{}
	}

	portMap := make(nat.PortSet)
	for port := range s.ContainerConfig.PortMapping {
		portMap[port] = struct{}{}
	}

	containerResponse, err := s.dockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image: s.ImageConfig.Image,
			Volumes: volumeMap,
			ExposedPorts: portMap,
			    AttachStdout: true,
			    AttachStderr: true,
		},
		&container.HostConfig{
			Mounts: s.ContainerConfig.VolumeMapping,
			RestartPolicy: container.RestartPolicy{
				Name:              container.RestartPolicyUnlessStopped,
				MaximumRetryCount: 5,
			},
			PortBindings: s.ContainerConfig.PortMapping,
		},
		nil, nil, "",
	)

	s.containerID = &containerResponse.ID
	return err
}
