// Package main implements the backend for nfinite.space service. It allows
// for users to upload their files, like in typical cloud storage, but nfinite.space
// leverages other user's disk to store shard of the original file.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

// Maps connection to key objects
var connections = map[*websocket.Conn]Client{}
var buffers = map[*websocket.Conn][]byte{}
var waitGroups = map[*websocket.Conn]*sync.WaitGroup{}

// Signleton instance for database, address flag
var database = NewDatabase()
var addr = flag.String("addr", "0.0.0.0:8080", "http service address")

// Singleton upgrader object
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

// Use the upgrader singleton above to upgrade regular connections to websockets
func upgradeToWebsocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgradeToWebsocket:", err)
		return nil, err
	}
	return c, err
}

// Gets the current websocket object for a given Client
func connForClient(c Client) *websocket.Conn {
	for con, cli := range connections {
		if cli.username == c.username {
			return con
		}
	}
	return nil
}

// Main listener function for an accepted connection
func listen(w http.ResponseWriter, r *http.Request) {
	c, err := upgradeToWebsocket(w, r)
	if err != nil {
		return
	}
	defer func() {
		c.Close()
	}()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Println("Message type is:", mt)
		if mt == websocket.TextMessage {
			log.Printf("recv: %s", message)

			var m map[string]interface{}
			if err = json.Unmarshal(message, &m); err != nil {
				log.Println("json unmarshal:", err)
				return
			}
			t := m["type"].(string)
			if t == "file" || t == "part" {
				handleFileUpload(m, c)
			} else if t == "registration" {
				handleRegistration(m, c)
			} else if t == "request" {
				handleFileRequest(m, c)
			} else {
				log.Println("type: unknown json type:", t)
			}
		} else {
			log.Println("Not text message, save to byte storage")
			buffers[c] = message
			if wt, ok := waitGroups[c]; ok {
				wt.Done()
			}
		}
	}
}

// Accept uploaded File over websocket c and then shard to peers
func handleFileUpload(m map[string]interface{}, c *websocket.Conn) {
	metadata := m["fileMeta"].(map[string]interface{})
	f := FileFromMetaData(metadata)
	f, err := getFileUpload(c, f)
	if err != nil {
		log.Println("couldn't get file upload", err)
		return
	}
	shardFile(f, c)
}

// Handle user's initial connection registration from websocket c
func handleRegistration(m map[string]interface{}, c *websocket.Conn) {
	metadata := m["userMeta"].(map[string]interface{})
	client := ClientFromMetaData(metadata)
	log.Println("Adding client", client)
	connections[c] = client
	database.AddClient(client)
	sendUsersFileMetaData(c)

	defer delete(connections, c)
}

// Handle request for a particular File
func handleFileRequest(m map[string]interface{}, c *websocket.Conn) {
	metadata := m["fileMeta"].(map[string]interface{})
	f := FileFromMetaData(metadata)
	f = database.GetFile(f.name, connections[c])
	reqs := database.FilePartRequestsForFile(f, connections[c])
	log.Println("Number of reqs:", len(reqs))
	for _, req := range reqs {
		var reqCon *websocket.Conn
		i := 0
		for reqCon == nil {
			if i >= len(req.owners) {
				// out of bounds
				log.Panicln("No available peers to fetch part from.")
				return
			}
			reqCon = connForClient(req.owners[i])
			i++
		}

		pt := fetchPart(reqCon, req.filePart)
		f.data = append(f.data, pt.data...)
	}
	log.Println("About to send file ", f.name)
	sendFileResponse(c, f)
}

// Send full File f to client via websocket c
func sendFileResponse(c *websocket.Conn, f File) {
	json := "{\"type\" : \"response\", \"fileMeta\" : { \"name\" : \"" + f.name + "\" } }"
	if err := c.WriteMessage(websocket.TextMessage, []byte(json)); err != nil {
		log.Println("send file response json: ", err)
		return
	}
	if err := c.WriteMessage(websocket.BinaryMessage, f.data); err != nil {
		log.Println("send user requested file:", err)
	}
}

// Provide the Client connected via websocket c a list of FileMetaData for the files they are storing.
// Sent when a connection is established and a Client can see what they've stored on nfinite.space.
func sendUsersFileMetaData(c *websocket.Conn) {
	json := "{ \"type\" : \"fileList\", \"files\" : [ "
	files := database.ClientsFiles(connections[c])
	for i, f := range files {
		json += " { \"fileMeta\" : { "
		json += "\"name\" : \"" + f.name + "\", \"lastModified\" : \"" + strconv.FormatInt(f.modified.Unix(), 10) + "\" } }"
		if i != len(files)-1 {
			json += ", "
		}
	}
	json += " ] }"
	if err := c.WriteMessage(websocket.TextMessage, []byte(json)); err != nil {
		log.Println("send users files metadata:", err)
	}
}

// Accepted the uploaded file and put the file data into File f
func getFileUpload(c *websocket.Conn, f File) (File, error) {
	mt, message, err := c.ReadMessage()
	if mt != websocket.BinaryMessage {
		log.Println("file upload: client tried to upload non-byte data:", mt, message)
		return File{}, errors.New("file upload: client tried to upload non-byte data")
	} else if err != nil {
		log.Println("file upload:", err)
		return File{}, err
	}
	log.Println("Client is", connections[c])
	if database.DoesFileExist(f, connections[c]) {
		return File{}, errors.New("File doesn't exist in database")
	}
	f.data = message
	database.InsertFile(f, connections[c])
	return f, nil
}

// Shard File f and distribute it round-robin style to connected Clients
func shardFile(f File, c *websocket.Conn) {
	splitAmount := len(f.data) / (len(connections) - 1)
	sp := splitAmount
	begin := 0
	i := 0
	log.Println("Length of data is", len(f.data))
	for con, cli := range connections {
		if con == c {
			continue
		}
		log.Println("Begin: ", begin, " splitAmt: ", splitAmount)
		fpData := f.data[begin:splitAmount]

		fp := FilePart{}
		fp.name = hash(fmt.Sprintf("%s%v", f.name, con))
		fp.parent = f
		fp.modified = f.modified
		fp.index = i
		fp.data = fpData

		log.Println("DEBUG: created fp: ", fp.name, fp.index, fp.parent.name)

		database.AddFilePart(fp, connections[c], cli)
		sendPart(con, fp)
		begin = splitAmount
		splitAmount += sp
		i++
	}
}

// Get the FilePart fp from client connected over websocket c.
// Use WaitGroup to hold until we've received the FilePart.
func fetchPart(c *websocket.Conn, fp FilePart) FilePart {
	json := "{\"type\" : \"request\", \"fileMeta\" : { \"name\" : \"" + fp.name + "\" } }"
	if err := c.WriteMessage(websocket.TextMessage, []byte(json)); err != nil {
		log.Println("send request json: ", err)
		return FilePart{}
	}
	log.Println("Sent request to client for part", fp.name)
	wt := sync.WaitGroup{}
	wt.Add(1)
	waitGroups[c] = &wt
	wt.Wait()
	log.Println("Got response from client for part", fp.name)
	delete(waitGroups, c)
	message := buffers[c]
	fp.data = message
	return fp
}

// Sends the provided FilePart f to the client connected over the websocket c
func sendPart(c *websocket.Conn, f FilePart) {
	json := "{\"type\" : \"part\", \"fileMeta\" : { \"name\" : \"" + f.name + "\", \"dateModified\" : \"" + strconv.FormatInt(f.modified.Unix(), 10) + "\" } }"
	log.Println("Sending json: ", json)
	if err := c.WriteMessage(websocket.TextMessage, []byte(json)); err != nil {
		log.Println("send part json: ", err)
		return
	}
	if err := c.WriteMessage(websocket.BinaryMessage, f.data); err != nil {
		log.Println("send part data: ", err)
		return
	}
}

// Sends the provided File f to the client connected over the websocket c
func sendFile(c *websocket.Conn, f File) {
	json := "{\"type\" : \"file\", \"fileMeta\" : { \"name\" : \"" + f.name + "\", \"dateModified\" : \"" + strconv.FormatInt(f.modified.Unix(), 10) + "\" } }"
	log.Println("Sending json: ", json)
	if err := c.WriteMessage(websocket.TextMessage, []byte(json)); err != nil {
		log.Println("send file json: ", err)
		return
	}
	if err := c.WriteMessage(websocket.BinaryMessage, f.data); err != nil {
		log.Println("send file data: ", err)
		return
	}
}

// Compiles the original file from the file parts, assumes fps is sorted by index
func sendFileFromParts(c *websocket.Conn, fps []FilePart, original File) {
	file := File{FileMetaData{original.name, original.modified}, []byte("")}
	for _, fp := range fps {
		file.data = append(file.data, fp.data...)
	}
	sendFile(c, file)
}

// Gets hash of provided string
func hash(s string) string {
	h := sha256.New()
	io.WriteString(h, s)
	return hex.EncodeToString(h.Sum(nil))
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/websockets", listen)
	log.Println("Now listening...")
	log.Fatal(http.ListenAndServe(*addr, nil))
}
