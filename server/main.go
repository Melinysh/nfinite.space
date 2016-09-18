package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// FileMetaData object
type FileMetaData struct {
	name     string
	modified time.Time
}

// File object
type File struct {
	FileMetaData
	data []byte
}

// FilePart object
type FilePart struct {
	File
	parent File
	index  int
}

//FilePartRequest object
type FilePartRequest struct {
	owners   []Client
	filePart FilePart
}

func fileFromMetaData(metadata map[string]interface{}) File {
	seconds, _ := strconv.ParseInt(metadata["dateModified"].(string), 10, 64)
	dateMod := time.Unix(seconds, 0)
	name := metadata["name"].(string)
	return File{FileMetaData{name, dateMod}, []byte("")}
}

// Client object
type Client struct {
	username string
	password string
}

func clientFromMetaData(metadata map[string]interface{}) Client {
	username := metadata["name"].(string)
	password := metadata["pass"].(string)
	password = hash(password)
	return Client{username, password}
}

// Maps connection to key objects
var connections = map[*websocket.Conn]Client{}
var buffers = map[*websocket.Conn][]byte{}
var waitGroups = map[*websocket.Conn]*sync.WaitGroup{}
var database = NewDatabase()
var addr = flag.String("addr", "0.0.0.0:8080", "http service address")

func connForClient(c Client) *websocket.Conn {
	for con, cli := range connections {
		if cli.username == c.username {
			return con
		}
	}
	return nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

func upgradeToWebsocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgradeToWebsocket:", err)
		return nil, err
	}
	return c, err
}

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
		log.Printf("recv: %s", message)
		log.Println("Message type is:", mt)
		if mt == websocket.TextMessage {
			var m map[string]interface{}
			if err = json.Unmarshal(message, &m); err != nil {
				log.Println("json unmarshal:", err)
				return
			}
			t := m["type"].(string)
			if t == "file" || t == "part" {
				metadata := m["fileMeta"].(map[string]interface{})
				f := fileFromMetaData(metadata)
				getFileUpload(c, f)
			} else if t == "registration" {
				metadata := m["userMeta"].(map[string]interface{})
				client := clientFromMetaData(metadata)
				connections[c] = client
				database.AddClient(client)
				sendUsersFileMetaData(c)

				defer delete(connections, c)
			} else if t == "request" {
				metadata := m["fileMeta"].(map[string]interface{})
				f := fileFromMetaData(metadata)
				f = database.GetFile(f.name, connections[c])
				reqs := database.FilePartRequestsForFile(f, connections[c])
				log.Println("Number of reqs:", len(reqs))
				for _, req := range reqs {
					var reqCon *websocket.Conn = nil
					i := 0
					for reqCon == nil {
						reqCon = connForClient(req.owners[i])
						i += 1
					}

					pt := fetchPart(reqCon, req.filePart)
					f.data = append(f.data, pt.data...)
				}
				log.Println("About to send file ", f.name)
				sendFileResponse(c, f)
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

func sendUsersFileMetaData(c *websocket.Conn) {
	/*	preamble := "{ \"type\" : \"fileList\" }"
		if err := c.WriteMessage(websocket.BinaryMessage, []byte(preamble)); err != nil {
			log.Println("send users files metadata preamble:", err)
		}*/
	json := "{ \"type\" : \"fileList\", \"files\" : [ "
	files := database.ClientsFiles(connections[c])
	for i, f := range files {
		json += "\"" + f.name + "\""
		if i != len(files)-1 {
			json += ", "
		}
	}
	json += " ] }"
	if err := c.WriteMessage(websocket.TextMessage, []byte(json)); err != nil {
		log.Println("send users files metadata:", err)
	}
}

func getFileUpload(c *websocket.Conn, f File) {
	mt, message, err := c.ReadMessage()
	if mt != websocket.BinaryMessage {
		log.Println("file upload: client tried to upload non-byte data:", mt, message)
		return
	} else if err != nil {
		log.Println("file upload:", err)
		return
	}
	if database.DoesFileExist(f, connections[c]) {
		return
	}
	f.data = message
	database.InsertFile(f, connections[c])
	splitAmount := len(f.data) / len(connections)
	begin := 0
	for con, cli := range connections {
		if con == c {
			continue
		}
		fpData := f.data[begin:splitAmount]

		fp := FilePart{}
		fp.name = hash(fmt.Sprintf("%s%v", f.name, con))
		fp.parent = f
		fp.modified = f.modified
		fp.index = begin / (len(f.data) / len(connections))
		fp.data = fpData

		log.Println("DEBUG: created fp: ", fp.name, fp.index, fp.parent.name)

		database.AddFilePart(fp, connections[c], cli)
		sendPart(con, fp)
		begin += splitAmount
		splitAmount *= 2
	}
}

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
