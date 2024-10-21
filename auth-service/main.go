package main

import (
	"log"
	"os"
	"strings"

	"github.com/gocql/gocql"
	"github.com/gofiber/fiber/v2"
)

var session *gocql.Session

func main() {
	// Initialize Cassandra session
	var err error
	session, err = initCassandra()
	if err != nil {
		log.Fatal("Failed to connect to Cassandra:", err)
	}
	defer session.Close()

	app := fiber.New()

	// Routes
	app.Post("/login", login)
	app.Post("/token/refresh", refreshToken)
	app.Post("/logout", logout)

	log.Fatal(app.Listen(":3000"))
}

func initCassandra() (*gocql.Session, error) {
	cassandraHosts := strings.Split(os.Getenv("CASSANDRA_HOSTS"), ",")
	cluster := gocql.NewCluster(cassandraHosts...)
	cluster.Keyspace = os.Getenv("CASSANDRA_KEYSPACE")
	cluster.Consistency = gocql.Quorum
	return cluster.CreateSession()
}
