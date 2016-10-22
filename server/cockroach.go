package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Database is a wrapper around the sql.DB object. To be used as a singleton
type Database struct {
	*sql.DB
}

// NewDatabase returns a new Database object. Should only be called once.
func NewDatabase() Database {
	db, err := sql.Open("postgres", "postgresql://root@localhost:26257?sslcert=%2Fhome%2Fubuntu%2Fnode1.cert&sslkey=%2Fhome%2Fubuntu%2Fnode1.key&sslmode=verify-full&sslrootcert=%2Fhome%2Fubuntu%2Fca.cert")
	if err != nil {
		log.Fatalln("database connection:", err)
	}
	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS nfinite"); err != nil {
		log.Fatal(err)
	}

	if _, err := db.Exec("SET DATABASE = nfinite"); err != nil {
		log.Fatal(err)
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS PartLookup (id SERIAL PRIMARY KEY, partId INT, ownerId INT);"); err != nil {
		log.Fatal(err)
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS FilePart (parentId INT, name string, id SERIAL PRIMARY KEY, fileIndex INT);"); err != nil {
		log.Fatal(err)
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS Client (id SERIAL, username string PRIMARY KEY, password string);"); err != nil {
		log.Fatal(err)
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS File (id SERIAL PRIMARY KEY, modified INT, name string, ownerId INT);"); err != nil {
		log.Fatal(err)
	}
	return Database{db}
}

// AddClient inserts Client c into database db
func (db *Database) AddClient(c Client) {
	if _, err := db.Exec("INSERT INTO Client (username, password) VALUES ($1, $2) ON CONFLICT (username) DO NOTHING", c.username, c.password); err != nil {
		log.Fatalln("insert client:", err)
	}
}

// ClientsFiles returns a slice of Files belonging to the Client c
func (db *Database) ClientsFiles(c Client) []File {
	dbFs := db.dbFilesForClient(c)
	var files []File
	for _, dbF := range dbFs {
		f := File{}
		f.name = dbF.name
		f.modified = time.Unix(int64(dbF.modified), 0)
		files = append(files, f)
	}
	return files
}

// GetFile returns a File from the database for a given name and Client c
func (db *Database) GetFile(name string, c Client) File {
	f := File{}
	f.name = name
	dbF := db.dbFileForClientFile(f, c)
	f.modified = time.Unix(int64(dbF.modified), 0)
	return f
}

// DoesFileExist checks if the File f exists for Client c
func (db *Database) DoesFileExist(f File, c Client) bool {
	dbC := db.dbClientForClient(c)
	var count uint64
	const countSQL = `
	SELECT COUNT(id) FROM File WHERE name=$1 AND ownerId=$2`
	if err := db.QueryRow(countSQL, f.name, dbC.id).Scan(&count); err != nil {
		log.Println("checking if file saved:", err)
		return false
	}
	return count > 0
}

// InsertFile inserts File f from Client c into the database
func (db *Database) InsertFile(f File, c Client) {
	dbC := db.dbClientForClient(c)
	db.insertFileForDbClient(f, dbC)
}

// AddFilePart inserts the FilePart fp for owner and storer. It will also update the file part lookup table.
func (db *Database) AddFilePart(fp FilePart, owner Client, storer Client) {
	dbC := db.dbClientForClient(storer)
	db.insertFilePart(fp, owner, dbC)
	dbFp := db.dbFilePartFromFilePart(fp)
	db.savePartLookup(dbFp, dbC)
}

// FilePartRequestsForFile returns a slice of FilePartRequests for a given Client c and File f
func (db *Database) FilePartRequestsForFile(f File, owner Client) []FilePartRequest {
	dbF := db.dbFileForClientFile(f, owner)
	dbFParts := db.dbFilePartsForDbFile(dbF)
	var reqs []FilePartRequest
	for _, p := range dbFParts {
		dbOwners := db.dbClientsForDbFilePart(p)
		var owners []Client
		for _, o := range dbOwners {
			owners = append(owners, Client{o.username, o.password})
		}
		fp := FilePart{parent: f, index: p.fileIndex}
		fp.name = p.name
		fp.modified = f.modified
		reqs = append(reqs, FilePartRequest{owners, fp})
	}
	return reqs
}

func (db *Database) dbClientForClient(c Client) DbClient {
	rows, err := db.Query("SELECT * FROM Client WHERE username=$1", c.username)
	if err != nil {
		log.Fatalln("Unable to get client for username", c.username, ":", err)
	}
	defer rows.Close()
	if !rows.Next() {
		log.Fatalln("No row for username", c.username)
	}
	return NewDbClient(rows)
}

func (db *Database) dbClientForID(id int) DbClient {
	rows, err := db.Query("SELECT * FROM Client WHERE id=$1", id)
	if err != nil {
		log.Fatalln("Unable to get client for id", id, ":", err)
	}
	defer rows.Close()
	if !rows.Next() {
		log.Fatalln("No row for id", id)
	}
	return NewDbClient(rows)
}

func (db *Database) insertFilePart(fp FilePart, owner Client, storer DbClient) {
	dbF := db.dbFileForClientFile(fp.parent, owner)
	if _, err := db.Exec("INSERT INTO FilePart (parentId, name, fileIndex) VALUES ($1, $2, $3)", dbF.id, fp.name, fp.index); err != nil {
		log.Println("save file part:", err)
	}
}

func (db *Database) dbFilePartFromFilePart(fp FilePart) DbFilePart {
	rows, err := db.Query("SELECT * FROM FilePart WHERE name=$1 AND fileIndex=$2", fp.name, fp.index)
	if err != nil {
		log.Fatalln("Unable to get file path for name", fp.name, ":", err)
	}
	defer rows.Close()
	if !rows.Next() {
		log.Fatalln("No row for file path", fp.name)
	}
	return NewDbFilePart(rows)

}

func (db *Database) dbFilePartsForDbFile(f DbFile) []DbFilePart {
	var parts []DbFilePart
	rows, err := db.Query("SELECT * FROM FilePart WHERE parentId=$1 ORDER BY fileIndex ASC", f.id)
	if err != nil {
		log.Fatalln("Unable to get file part for file id", f.id, ":", err)
	}
	defer rows.Close()
	for rows.Next() {
		parts = append(parts, NewDbFilePart(rows))
	}
	return parts
}

func (db *Database) savePartLookup(dbFp DbFilePart, dbC DbClient) {
	if _, err := db.Exec("INSERT INTO PartLookup (partId, ownerId) VALUES ($1, $2)", dbFp.id, dbC.id); err != nil {
		log.Println("save part lookup:", err)
	}
}

func (db *Database) dbClientsForDbFilePart(dbFp DbFilePart) []DbClient {
	rows, err := db.Query("SELECT ownerId FROM PartLookup WHERE partId=$1", dbFp.id)
	if err != nil {
		log.Fatalln("Unable to get file for name", dbFp.name, ":", err)
	}
	//	defer rows.Close()
	var dbClients []DbClient
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Println("client from file part lookup:", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		dbClients = append(dbClients, db.dbClientForID(id))
	}

	return dbClients
}

func (db *Database) dbFilesForClient(owner Client) []DbFile {
	dbC := db.dbClientForClient(owner)
	rows, err := db.Query("SELECT * FROM File WHERE ownerId=$1", dbC.id)
	if err != nil {
		log.Fatalln("Unable to get file for owner", owner.username, ":", err)
	}
	defer rows.Close()
	var dbFiles []DbFile
	for rows.Next() {
		dbFiles = append(dbFiles, NewDbFile(rows))
	}
	return dbFiles

}

func (db *Database) dbFileForClientFile(f File, c Client) DbFile {
	dbC := db.dbClientForClient(c)
	rows, err := db.Query("SELECT * FROM File WHERE name=$1 AND ownerId=$2", f.name, dbC.id)
	if err != nil {
		log.Fatalln("Unable to get file for name", f.name, ":", err)
	}
	defer rows.Close()
	if !rows.Next() {
		log.Fatalln("No row for file", f.name)
	}
	return NewDbFile(rows)
}

func (db *Database) insertFileForDbClient(f File, dbC DbClient) {
	if _, err := db.Exec("INSERT INTO File (modified, name, ownerId) VALUES ($1, $2, $3)", f.modified.Unix(), f.name, dbC.id); err != nil {
		log.Println("insert file for db client:", err)
	}
}
