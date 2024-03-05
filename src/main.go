package main

import (
	"context"
	"fmt"
	"log/slog"
	"main/services"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
)

func cerr(err error) {
	if err != nil {
		panic(err)
	}
}

func Jellyfin(ctx context.Context, cli *client.Client) {
	jellyfin, err := services.NewJellyfin(ctx, cli)
	cerr(err)
	cerr(jellyfin.Start(ctx))
	cerr(jellyfin.BaseSetup())
}

func Jellyseerr(ctx context.Context, wg *sync.WaitGroup, cli *client.Client) {
	wg.Add(1)
	jellyseerr, err := services.NewJellyseerr(ctx, cli)
	cerr(err)
	cerr(jellyseerr.Start(ctx))
	go jellyseerr.AttachLogs(ctx)
	// cerr(jellyseerr.BaseSetup())
}

func main() {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(h))

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	cerr(err)
	defer cli.Close()

	// Jellyfin(ctx, cli)

	appUUID := uuid.New()

	networkCreateResp, err := cli.NetworkCreate(ctx, fmt.Sprintf("named-network-%s", appUUID.String()), types.NetworkCreate{
		Driver:     "bridge",
		Attachable: true,
		Options: map[string]string{
			"com.docker.network.bridge.enable_icc": "enable", // option not recognized
		},
	})
	cerr(err)

	slog.Debug(networkCreateResp.ID)

	wg := sync.WaitGroup{}
	Jellyseerr(ctx, &wg, cli)

	wg.Wait()
	slog.Debug("meow")
}
