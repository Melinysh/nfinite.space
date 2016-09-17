package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type dbClient struct {
	id       int
	username string
	password string
}

func newDbClient(r *sql.Rows) dbClient {
	var id int
	var username, password string
	if err := r.Scan(&id, &username, &password); err != nil {
		log.Println("new db client:", err)
	}
	return dbClient{id, username, password}
}

func (db *Database) dbClientForClient(c Client) dbClient {
	rows, err := db.Query("SELECT * FROM Client WHERE username=?", c.username)
	if err != nil {
		log.Fatalln("Unable to get client for username", c.username, ":", err)
	}
	defer rows.Close()
	if !rows.Next() {
		log.Fatalln("No row for username", c.username)
	}
	return newDbClient(rows)
}

func (db *Database) AddClient(c Client) {
	if _, err := db.Exec("INSERT OR REPLACE INTO Client (username, password) VALUES ($1, $2)", c.username, c.password); err != nil {
		log.Fatalln("upsert client:", err)
	}
}

type dbFilePart struct {
	parentId int
	name     string
	lookupId int
}

func newDbFilePart(r *sql.Rows) dbFilePart {
	var parentId, lookupId int
	var name string
	if err := r.Scan(&parentId, &name, &lookupId); err != nil {
		log.Println("new db file part:", err)
	}
	return dbFilePart{parentId, name, lookupId}
}

type dbFileLookup struct {
	id      int
	partId  int
	ownerId int
}

func newDbFileLookup(r *sql.Rows) dbFileLookup {
	var id, parentId, ownerId int
	if err := r.Scan(&id, &parentId, &ownerId); err != nil {
		log.Println("new db file lookup:", err)
	}
	return dbFileLookup{id, parentId, ownerId}
}

type dbFile struct {
	id       int
	modified int
	name     string
	ownerId  string
}

func newDbFile(r *sql.Rows) dbFile {
	var id, modified int
	var name, ownerId string
	if err := r.Scan(&id, &modified, &name, &ownerId); err != nil {
		log.Println("new db file:", err)
	}
	return dbFile{id, modified, name, ownerId}
}

// Database object, only one should be used
type Database struct {
	*sql.DB
}

func NewDatabase() Database {
	db, err := sql.Open("nfinite.db", "postgresql://root@localhost:26257?sslmode=disable")
	if err != nil {
		log.Fatalln("database connection:", err)
	}
	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS nfinite"); err != nil {
		log.Fatal(err)
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS PartLookup (id INT PRIMARY KEY, partId INT, ownerId INT);"); err != nil {
		log.Fatal(err)
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS FilePart (parentId INT, name string, lookupId INT);"); err != nil {
		log.Fatal(err)
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS Client (id INT PRIMARY KEY, username string, password string);"); err != nil {
		log.Fatal(err)
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS File (id INT PRIMARY KEY, modified INT, name string, ownerId INT);"); err != nil {
		log.Fatal(err)
	}
	return Database{db}
}

func (db *Database) savePartLookup(fp FilePart, c Client) {
	if _, err := db.Exec("INSERT INTO PartLookup (partId, ownerId) VALUES ($1, $2)"); err != nil {
		log.Println("save part lookup:", err)
	}
}
