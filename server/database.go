package main

import (
	"database/sql"
	"log"
)

// DbFile is a database representation of a File
type DbFile struct {
	id       int
	modified int
	name     string
	ownerID  string
}

// NewDbFile returns a new DbFile for the results found in the provided sql.Rows
func NewDbFile(r *sql.Rows) DbFile {
	var id, modified int
	var name, ownerID string
	if err := r.Scan(&id, &modified, &name, &ownerID); err != nil {
		log.Println("new db file:", err)
	}
	return DbFile{id, modified, name, ownerID}
}

// DbFilePart is a database representation of a FilePart
type DbFilePart struct {
	parentID  int
	name      string
	id        int
	fileIndex int
}

// NewDbFilePart creates a new DbFilePart from the sql.Rows provided
func NewDbFilePart(r *sql.Rows) DbFilePart {
	var parentID, id, fileIndex int
	var name string
	if err := r.Scan(&parentID, &name, &id, &fileIndex); err != nil {
		log.Println("new db file part:", err)
	}
	return DbFilePart{parentID, name, id, fileIndex}
}

// DbFileLookup represents the many-to-many relationship between FileParts and Clients
type DbFileLookup struct {
	id      int
	partID  int
	ownerID int
}

// NewDbFileLookup creates a new DbFileLookup from the sql.Rows provided
func NewDbFileLookup(r *sql.Rows) DbFileLookup {
	var id, parentID, ownerID int
	if err := r.Scan(&id, &parentID, &ownerID); err != nil {
		log.Println("new db file lookup:", err)
	}
	return DbFileLookup{id, parentID, ownerID}
}

// DbClient is the database representation of a Client
type DbClient struct {
	id       int
	username string
	password string
}

// NewDbClient creates a new DbCLient from the sql.Rows provided
func NewDbClient(r *sql.Rows) DbClient {
	var id int
	var username, password string
	if err := r.Scan(&id, &username, &password); err != nil {
		log.Println("new db client:", err)
	}
	return DbClient{id, username, password}
}
