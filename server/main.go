package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// FileMetaData
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

// Maps client username to connection
var connections = map[string]*websocket.Conn{}
var database = NewDatabase()
var addr = flag.String("addr", "0.0.0.0:8080", "http service address")

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
				connections[client.username] = c
				sendUsersFileMetaData(c)
				database.AddClient(client)
				defer delete(connections, client)
			} else {
				log.Println("type: unknown json type:", t)
			}
		} else {
			log.Println("Not text message")
		}
	}
}

func sendUsersFileMetaData(c *websocket.Conn) {
	json := "{ \"files\" : [ \"hello.txt\", \"lol.jpg\"] }" // TODO: fetch user's actual files.
	if err := c.WriteMessage(websocket.BinaryMessage, []byte(json)); err != nil {
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
	f.data = message
	err = ioutil.WriteFile(f.name, f.data, 0644)
	if err != nil {
		log.Println("write file:", err)
		return
	}
	log.Println("DEBUG: wrote file to ./", f.name)
	f.data = f.data[0 : len(f.data)/2]
	time.Sleep(time.Second * 2)
	sendPart(c, f)
}

func requestPart(c *websocket.Conn, fp FilePart) {
	json := "{\"type\" : \"request\", \"fileMeta\" : { \"name\" : \"" + fp.name + "\" } }"
	if err := c.WriteMessage(websocket.BinaryMessage, []byte(json)); err != nil {
		log.Println("send request json: ", err)
		return
	}
}

func sendPart(c *websocket.Conn, f File) {
	json := "{\"type\" : \"part\", \"fileMeta\" : { \"name\" : \"" + hash(f.name) + "\", \"dateModified\" : \"" + strconv.FormatInt(f.modified.Unix(), 10) + "\" } }"
	log.Println("Sending json: ", json)
	if err := c.WriteMessage(websocket.BinaryMessage, []byte(json)); err != nil {
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
	if err := c.WriteMessage(websocket.BinaryMessage, []byte(json)); err != nil {
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
	log.Fatal(http.ListenAndServe(*addr, nil))
}
