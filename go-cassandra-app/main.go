package main

import (
	"fmt"
	"log"

	"github.com/gocql/gocql"
)

func main() {
	// Create a cluster configuration
	cluster := gocql.NewCluster("127.0.0.1") // Cassandra container runs on localhost in development
	cluster.Keyspace = "my_keyspace"         // Default keyspace
	cluster.Consistency = gocql.Quorum

	// Create a new session
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatal("Failed to connect to Cassandra:", err)
	}
	defer session.Close()

	// Execute a simple query
	iter := session.Query("SELECT keyspace_name FROM system_schema.keyspaces").Iter()

	// Print keyspaces
	fmt.Println("Existing keyspaces:")
	var keyspaceName string
	for iter.Scan(&keyspaceName) {
		fmt.Println(keyspaceName)
	}

	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}
}
