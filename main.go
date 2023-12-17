package main

import (
	"data-replication/database"
	"data-replication/service"
	"data-replication/util"
	"database/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"log"
)

const (
	masterDSN  = "user=postgres password=Master123 host=localhost port=5445 dbname=employee sslmode=disable"
	replicaDSN = "user=postgres password=Replica123 host=localhost port=5446 dbname=employee sslmode=disable"
)

var (
	masterDB  *sql.DB
	replicaDB *sql.DB
)

func main() {
	var err error

	// Initialize master and replica databases
	masterDB, err = database.InitDB(masterDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer masterDB.Close()

	replicaDB, err = database.InitDB(replicaDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer replicaDB.Close()

	dataReplicationService := service.InitDataReplicationService(masterDB, replicaDB)
	// Start a goroutine to periodically check replica availability
	go util.CheckReplicaAvailability(replicaDB)

	// Create Gin router
	router := gin.Default()

	// Define API routes
	router.GET("/employees", dataReplicationService.GetEmployees)
	router.POST("/employees", dataReplicationService.CreateEmployee)

	// Run the server
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
