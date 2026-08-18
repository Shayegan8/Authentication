package module

import (
	"myblog"

	"github.com/redis/go-redis/v9"
)

func InitRDB() *redis.Client {
	opt, err := redis.ParseURL(myblog.Config["redis"])
	if err != nil {
		panic(err)
	}
	return redis.NewClient(opt)
}
