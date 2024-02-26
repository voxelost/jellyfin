package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"main/cache"
	"os"

	"github.com/docker/docker/client"
)


func cerr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	cerr(err)
	defer cli.Close()

	dbCache, err := cache.New()
	cerr(err)

	jellyfin, err := NewJellyfin(cli, dbCache)
	cerr(err)

	err = jellyfin.Start(ctx)
	cerr(err)

	reader, err := jellyfin.Logs(ctx)
	cerr(err)

	defer reader.Close()

	hdr := make([]byte, 8)
    for {
        _, err := reader.Read(hdr)
        if err != nil {
            log.Fatal(err)
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
        fmt.Fprint(w, string(dat))
    }
}
