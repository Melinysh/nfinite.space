package main

import (
	"database/sql"
	"log"
	"time"

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
	rows, err := db.Query("SELECT * FROM Client WHERE username=$1", c.username)
	if err != nil {
		log.Fatalln("Unable to get client for username", c.username, ":", err)
	}
	defer rows.Close()
	if !rows.Next() {
		log.Fatalln("No row for username", c.username)
	}
	return newDbClient(rows)
}

func (db *Database) dbClientForId(id int) dbClient {
	rows, err := db.Query("SELECT * FROM Client WHERE id=$1", id)
	if err != nil {
		log.Fatalln("Unable to get client for id", id, ":", err)
	}
	defer rows.Close()
	if !rows.Next() {
		log.Fatalln("No row for id", id)
	}
	return newDbClient(rows)
}

type dbFilePart struct {
	parentId  int
	name      string
	id        int
	fileIndex int
}

func (db *Database) insertFilePart(fp FilePart, owner Client, storer dbClient) {
	dbF := db.dbFileForClientFile(fp.parent, owner)
	if _, err := db.Exec("INSERT INTO FilePart (parentId, name, fileIndex) VALUES ($1, $2, $3)", dbF.id, fp.name, fp.index); err != nil {
		log.Println("save file part:", err)
	}
}

func (db *Database) dbFilePartFromFilePart(fp FilePart) dbFilePart {
	rows, err := db.Query("SELECT * FROM FilePart WHERE name=$1 AND fileIndex=$2", fp.name, fp.index)
	if err != nil {
		log.Fatalln("Unable to get file path for name", fp.name, ":", err)
	}
	defer rows.Close()
	if !rows.Next() {
		log.Fatalln("No row for file path", fp.name)
	}
	return newDbFilePart(rows)

}

func newDbFilePart(r *sql.Rows) dbFilePart {
	var parentId, id, fileIndex int
	var name string
	if err := r.Scan(&parentId, &name, &id, &fileIndex); err != nil {
		log.Println("new db file part:", err)
	}
	return dbFilePart{parentId, name, id, fileIndex}
}

func (db *Database) dbFilePartsForDbFile(f dbFile) []dbFilePart {
	var parts []dbFilePart
	rows, err := db.Query("SELECT * FROM FilePart WHERE parentId=$1 ORDER BY fileIndex ASC", f.id)
	if err != nil {
		log.Fatalln("Unable to get file part for file id", f.id, ":", err)
	}
	defer rows.Close()
	for rows.Next() {
		parts = append(parts, newDbFilePart(rows))
	}
	return parts
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

func (db *Database) savePartLookup(dbFp dbFilePart, dbC dbClient) {
	if _, err := db.Exec("INSERT INTO PartLookup (partId, ownerId) VALUES ($1, $2)", dbFp.id, dbC.id); err != nil {
		log.Println("save part lookup:", err)
	}
}

func (db *Database) dbClientsForDbFilePart(dbFp dbFilePart) []dbClient {
	rows, err := db.Query("SELECT ownerId FROM PartLookup WHERE partId=$1", dbFp.id)
	if err != nil {
		log.Fatalln("Unable to get file for name", dbFp.name, ":", err)
	}
	//	defer rows.Close()
	var dbClients []dbClient
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
		dbClients = append(dbClients, db.dbClientForId(id))
	}

	return dbClients
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

func (db *Database) dbFilesForClient(owner Client) []dbFile {
	dbC := db.dbClientForClient(owner)
	rows, err := db.Query("SELECT * FROM File WHERE ownerId=$2", dbC.id)
	if err != nil {
		log.Fatalln("Unable to get file for owner", owner.username, ":", err)
	}
	defer rows.Close()
	var dbFiles []dbFile
	for rows.Next() {
		dbFiles = append(dbFiles, newDbFile(rows))
	}
	return dbFiles

}

func (db *Database) dbFileForClientFile(f File, c Client) dbFile {
	dbC := db.dbClientForClient(c)
	rows, err := db.Query("SELECT * FROM File WHERE name=$1 AND ownerId=$2", f.name, dbC.id)
	if err != nil {
		log.Fatalln("Unable to get file for name", f.name, ":", err)
	}
	defer rows.Close()
	if !rows.Next() {
		log.Fatalln("No row for file", f.name)
	}
	return newDbFile(rows)
}

func (db *Database) insertFileForDbClient(f File, dbC dbClient) {
	if _, err := db.Exec("INSERT INTO File (modified, name, ownerId) VALUES ($1, $2, $3)", f.modified.Unix(), f.name, dbC.id); err != nil {
		log.Println("insert file for db client:", err)
	}
}

// Database object, only one should be used
type Database struct {
	*sql.DB
}

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

func (db *Database) AddClient(c Client) {
	if _, err := db.Exec("INSERT INTO Client (username, password) VALUES ($1, $2) ON CONFLICT (username) DO NOTHING", c.username, c.password); err != nil {
		log.Fatalln("insert client:", err)
	}
}
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

func (db *Database) GetFile(name string, c Client) File {
	f := File{}
	f.name = name
	dbF := db.dbFileForClientFile(f, c)
	f.modified = time.Unix(int64(dbF.modified), 0)
	return f
}

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

func (db *Database) InsertFile(f File, c Client) {
	dbC := db.dbClientForClient(c)
	db.insertFileForDbClient(f, dbC)
}

func (db *Database) AddFilePart(fp FilePart, owner Client, storer Client) {
	dbC := db.dbClientForClient(storer)
	db.insertFilePart(fp, owner, dbC)
	dbFp := db.dbFilePartFromFilePart(fp)
	db.savePartLookup(dbFp, dbC)
}

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
