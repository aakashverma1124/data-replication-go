package service

import (
	"data-replication/model"
	"database/sql"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type DataReplication struct {
	MasterDB  *sql.DB
	ReplicaDB *sql.DB
}

func InitDataReplicationService(masterDB *sql.DB, replicaDB *sql.DB) *DataReplication {
	return &DataReplication{
		MasterDB:  masterDB,
		ReplicaDB: replicaDB,
	}
}

func (dataReplicationService *DataReplication) CreateEmployee(c *gin.Context) {
	// Begin a transaction on the master database
	txMaster, err := dataReplicationService.MasterDB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() {
		// Rollback the transaction if there is an error
		if err := recover(); err != nil {
			txMaster.Rollback()
		}
	}()

	// Begin a transaction on the replica database
	txReplica, err := dataReplicationService.ReplicaDB.Begin()
	if err != nil {
		txMaster.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() {
		// Rollback the transactions if there is an error
		if err := recover(); err != nil {
			txMaster.Rollback()
			txReplica.Rollback()
		}
	}()

	var employee model.Employee
	if err := c.BindJSON(&employee); err != nil {
		txMaster.Rollback()
		txReplica.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Write to master database within the transaction
	resultMaster, err := txMaster.Exec("INSERT INTO emp (id, name, salary) VALUES ($1, $2, $3) RETURNING id", employee.ID, employee.Name, employee.Salary)
	if err != nil {
		txMaster.Rollback()
		txReplica.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Retrieve the last inserted ID from master
	_, err = resultMaster.RowsAffected()
	if err != nil {
		txMaster.Rollback()
		txReplica.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Write to replica database within the transaction
	if _, err := txReplica.Exec("INSERT INTO emp (id, name, salary) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET name = $2, salary = $3", employee.ID, employee.Name, employee.Salary); err != nil {
		txMaster.Rollback()
		txReplica.Rollback()
		log.Println("Failed to sync data to replica:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Commit the transactions only if both succeed
	if err := txMaster.Commit(); err != nil {
		txReplica.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := txReplica.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the inserted ID in the response
	c.JSON(http.StatusOK, gin.H{"id": employee.ID, "message": "Employee created successfully"})
}

func (dataReplicationService *DataReplication) GetEmployees(c *gin.Context) {
	rows, err := dataReplicationService.ReplicaDB.Query("SELECT id, name, salary FROM emp")
	if err != nil {
		// If the replica is not available, switch to the master database
		log.Println("Replica is not available. Reading from master.")
		rows, err = dataReplicationService.MasterDB.Query("SELECT id, name, salary FROM emp")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
	}

	var employees []model.Employee
	for rows.Next() {
		var employee model.Employee
		if err := rows.Scan(&employee.ID, &employee.Name, &employee.Salary); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		employees = append(employees, employee)
	}

	c.JSON(http.StatusOK, employees)
}
