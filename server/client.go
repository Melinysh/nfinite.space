package main

// Client represents our basic user object
type Client struct {
	username string
	password string
}

// ClientFromMetaData creates a new Client from the provided metadata
func ClientFromMetaData(metadata map[string]interface{}) Client {
	username := metadata["name"].(string)
	password := metadata["pass"].(string)
	password = hash(password)
	return Client{username, password}
}
