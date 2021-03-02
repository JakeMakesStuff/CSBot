package db

import (
	"context"
	"github.com/jackc/pgx/v4"
	"os"
)

// Conn is used to define the database connection.
var Conn *pgx.Conn

func init() {
	var err error
	if Conn, err = pgx.Connect(context.TODO(), os.Getenv("CONNECTION_STRING")); err != nil {
		panic(err)
	}
	if err = Conn.Ping(context.TODO()); err != nil {
		panic(err)
	}
}
