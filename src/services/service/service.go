package service

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"main/utils"
	"net/url"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type ImageConfig struct {
	RefString        string
	Image            string
	ImagePullOptions types.ImagePullOptions
}

type ContainerConfig struct {
	VolumeMapping []mount.Mount // todo: automate
	Env           []string

	portMapping map[nat.Port]nat.Port // Host Port -> Container Port
}

type Service struct {
	ImageConfig     ImageConfig
	ContainerConfig ContainerConfig

	dockerClient *client.Client
	// cacheClient  *cache.RoseDBClient
	containerID *string   // Docker Container ID of the Service
	uuid        uuid.UUID // Internal ID
}

func NewService(ctx context.Context, imageConfig ImageConfig, containerConfig ContainerConfig, dockerClient *client.Client, containerPorts []int) (*Service, error) {
	uuid := uuid.New()

	var patchedVolumes []mount.Mount
	for _, vol := range containerConfig.VolumeMapping {
		if vol.Type == "volume" {
			vol.Source = fmt.Sprintf("%s-%s", vol.Source, uuid)
			patchedVolumes = append(patchedVolumes, vol)
		}
	}

	containerConfig.VolumeMapping = patchedVolumes

	portMapping := make(map[nat.Port]nat.Port)
	for _, p := range containerPorts {
		hostPortAddress, err := utils.GetFreePort(ctx)
		if err != nil {
			return nil, err
		}

		hostPort, err := nat.NewPort("tcp", fmt.Sprintf("%d", hostPortAddress))
		if err != nil {
			return nil, err
		}

		containerPort, err := nat.NewPort("tcp", fmt.Sprintf("%d", p))
		if err != nil {
			return nil, err
		}

		portMapping[hostPort] = containerPort
	}

	containerConfig.portMapping = portMapping

	return &Service{
		ImageConfig:     imageConfig,
		ContainerConfig: containerConfig,
		dockerClient:    dockerClient,
		// cacheClient:     cacheClient,
		uuid: uuid,
	}, nil
}

// Docker Container ID of the Service
func (s *Service) ID() (string, error) {
	if s.containerID == nil {
		return "", fmt.Errorf("service not created yet")
	}

	return *s.containerID, nil
}

func (s *Service) ApiPort() (nat.Port, error) {
	// todo: check if any of the services needs more than a single port exposed
	for k := range s.ContainerConfig.portMapping {
		return k, nil
	}

	return "", fmt.Errorf("service doesn't expose any ports")
}

func (s *Service) ApiAddress() (url.URL, error) {
	port, err := s.ApiPort()
	if err != nil {
		return url.URL{}, err
	}

	return url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%s", port.Port()),
	}, nil
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

func (s *Service) EnsureContainer(ctx context.Context) error {
	volumeMap := make(map[string]struct{})
	for _, volume := range s.ContainerConfig.VolumeMapping {
		volumeMap[volume.Target] = struct{}{}
	}

	exposedPortsMap := make(nat.PortSet)
	portBindings := make(nat.PortMap)
	for hostPort, containerPort := range s.ContainerConfig.portMapping {
		exposedPortsMap[hostPort] = struct{}{}

		newBinding := nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: hostPort.Port(),
		}

		if _, ok := portBindings[hostPort]; ok {
			portBindings[containerPort] = append(portBindings[containerPort], newBinding)
		} else {
			portBindings[containerPort] = []nat.PortBinding{newBinding}
		}
	}

	containerResponse, err := s.dockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image:        s.ImageConfig.Image,
			Volumes:      volumeMap,
			ExposedPorts: exposedPortsMap,
			AttachStdout: true,
			AttachStderr: true,
			Env:          s.ContainerConfig.Env,
		},
		&container.HostConfig{
			Mounts: s.ContainerConfig.VolumeMapping,
			RestartPolicy: container.RestartPolicy{
				Name:              container.RestartPolicyUnlessStopped,
				MaximumRetryCount: 5,
			},
			PortBindings: portBindings,
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				"default": &network.EndpointSettings{},
			},
		}, nil, "",
	)

	s.containerID = &containerResponse.ID
	return err
}

func (s *Service) Start(ctx context.Context) error {
	err := s.EnsureImage(ctx)
	if err != nil {
		return err
	}

	err = s.EnsureContainer(ctx)
	if err != nil {
		return err
	}

	containerID, err := s.ID()
	if err != nil {
		return err
	}

	return s.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (s *Service) GetLogsReader(ctx context.Context) (io.ReadCloser, error) {
	containerId, err := s.ID()
	if err != nil {
		return nil, err
	}

	return s.dockerClient.ContainerLogs(ctx, containerId, container.LogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: false,
		Follow:     true,
		Tail:       "40",
	})
}

func (s *Service) AttachLogs(ctx context.Context) error {
	reader, err := s.GetLogsReader(ctx)
	if err != nil {
		return err
	}

	defer reader.Close()

	hdr := make([]byte, 8)
	for {
		_, err := reader.Read(hdr)
		if err != nil {
			return err
		}
		var w io.Writer
		switch hdr[0] {
		case 1:
			w = os.Stdout
		default:
			w = os.Stderr
		}
		count := binary.BigEndian.Uint32(hdr[4:])
		dat := make([]byte, count)
		_, err = reader.Read(dat)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		fmt.Fprint(w, string(dat))
	}

	return nil
}
