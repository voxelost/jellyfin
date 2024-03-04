package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"main/cache"
	"os"
	"time"

	"github.com/docker/docker/client"
)

func cerr(err error) {
	if err != nil {
		panic(err)
	}
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

	dbCache, err := cache.New()
	cerr(err)

	jellyfin, err := NewJellyfin(ctx, cli, dbCache)
	cerr(err)

	err = jellyfin.Start(ctx)
	cerr(err)

	exposedPort, err := jellyfin.ApiPort()
	cerr(err)

	slog.DebugContext(ctx, fmt.Sprintf("exposed port is %s", exposedPort.Port()))

	go jellyfin.AttachLogs(ctx)
	time.Sleep(5 * time.Second)

	jellyfin.BaseSetup()

	// `X-Emby-Authorization:
	// MediaBrowser Client="Jellyfin Web", Device="Firefox", DeviceId="TW96aWxsYS81LjAgKE1hY2ludG9zaDsgSW50ZWwgTWFjIE9TIFggMTAuMTU7IHJ2OjEyMy4wKSBHZWNrby8yMDEwMDEwMSBGaXJlZm94LzEyMy4wfDE3MDk1MTM4NzIxOTQ1", Version="10.8.13"`

	resp, err := jellyfin.HttpPost("/Users/authenticatebyname", `{"Username":"root","Pw":"root"}`)
	cerr(err)

	type tokenWrapper struct {
		Token string `json:"AccessToken"`
	}

	tw := tokenWrapper{}

	buf := make([]byte, resp.ContentLength)
	defer resp.Body.Close()

	resp.Body.Read(buf)
	json.Unmarshal(buf, &tw)

	slog.Debug(fmt.Sprintf("got token: %s", tw.Token))

	slog.Info("meow")
}
