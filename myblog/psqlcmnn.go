package myblog

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

var Postgres_client *pgx.Conn

func InitPDB() {
	log.Println("Initializing postgresql")
	db, err := pgx.Connect(context.Background(), "postgres://shayegan8/test?host=/var/run/postgresql")
	if err != nil {
		panic(err)
	}
	Postgres_client = db
	log.Println("Initializing postgresql completed")
}
