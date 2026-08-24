package myblog

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var Redis_client *redis.Client

func InitRDB() {
	log.Println("Initializing Redis")
	Redis_client = redis.NewClient(&redis.Options{
		Network:  "unix",
		Addr:     "/run/redis/redis-server.sock", // the permission for this file should be 400 and this apis are getting run with systemd/docker images by sudo user itself
		PoolSize: 30,
	})
	res := Redis_client.Ping(context.Background())
	res.Name()
	log.Println("initializing Redis finished")
}
