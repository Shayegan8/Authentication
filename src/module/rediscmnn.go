package module

import (
	"os"

	"github.com/redis/go-redis/v9"
)

func InitRDB() *redis.Client {
	return redis.NewClient(&redis.Options{
		Network:  "unix",
		Addr:     os.ExpandEnv("$HOME/redissock.sock"), // the permission for this file should be 400 and this apis are getting run with systemd/docker images by sudo user itself
		PoolSize: 30,
	})
}
