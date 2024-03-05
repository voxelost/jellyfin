package cache

import (
	"encoding/json"
	"fmt"

	"github.com/flower-corp/rosedb"
)

type RoseDBClient struct {
	client *rosedb.RoseDB
}

func New() (*RoseDBClient, error) {
	cli, err := rosedb.Open(rosedb.DefaultOptions("./.dev/tmp/cache"))
	if err != nil {
		return nil, err
	}

	return &RoseDBClient{
		client: cli,
	}, nil
}

func (client *RoseDBClient) Set(key string, value any) error {
	var val []byte
	var err error

	switch cast := value.(type) {
	case []byte:
		val = cast
	case string:
		val = []byte(cast)
	case int:
		val = []byte(fmt.Sprintf("%d", value))
	default:
		val, err = json.Marshal(value)
		if err != nil {
			return err
		}
	}

	return client.client.Set([]byte(key), val)
}

func (client *RoseDBClient) Get(key string) ([]byte, error) {
	return client.client.Get([]byte(key))
}

func (client *RoseDBClient) GetString(key string) (string, error) {
	value, err := client.Get(key)
	return string(value), err
}

func (client *RoseDBClient) GetObject(key string, v any) error {
	value, err := client.Get(key)
	if err != nil {
		return err
	}

	return json.Unmarshal(value, v)
}
