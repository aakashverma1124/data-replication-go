package util

import (
	"data-replication/database"
	"database/sql"
	"log"
	"time"
)

func CheckReplicaAvailability(replicaDB *sql.DB) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := database.PingDB(replicaDB); err != nil {
				log.Println("Replica is down.")
			}
		}
	}
}
