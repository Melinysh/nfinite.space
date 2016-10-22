package main

import (
	"strconv"
	"time"
)

// FileMetaData holds user facing info for a File
type FileMetaData struct {
	name     string
	modified time.Time
}

// File is composed of metadata and the raw file's bytes
type File struct {
	FileMetaData
	data []byte
}

// FilePart is a special File that is created from sharding another File
type FilePart struct {
	File
	parent File
	index  int
}

//FilePartRequest represents a request for a File Part
type FilePartRequest struct {
	owners   []Client
	filePart FilePart
}

// FileFromMetaData gets the File for the provided metadata
func FileFromMetaData(metadata map[string]interface{}) File {
	seconds, _ := strconv.ParseInt(metadata["dateModified"].(string), 10, 64)
	dateMod := time.Unix(seconds/1000, 0)
	name := metadata["name"].(string)
	return File{FileMetaData{name, dateMod}, []byte("")}
}
