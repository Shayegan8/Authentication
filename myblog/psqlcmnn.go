package myblog

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Postgres_client *pgxpool.Pool

func InitPDB() {
	log.Println("Initializing postgresql")
	db, err := pgxpool.New(context.Background(), "postgres://shayegan8/test?host=/var/run/postgresql")
	if err != nil {
		panic(err)
	}
	Postgres_client = db
	log.Println("Initializing postgresql completed")
}
