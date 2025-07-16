package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

var _ = net.Listen
var _ = os.Exit

func getUrl (conn net.Conn, mapUrls map[string]string) {
	//make a buffer and read the request
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Failed to read the buffer")
		os.Exit(1)
	}

	//change the buffer to a string, get the url and check if it is there
	stringurl := string(buffer)
	parts := strings.Split(stringurl, "\r\n")
	url := (strings.Split(parts[0], " "))[1]

	//check if url is in the set of urls
	_ , exists := mapUrls[url]
	if exists {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}

func addUrl (mapUrls map[string]string, url string, val string) {
	mapUrls[url] = val
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	fmt.Println("Accepted conection from: ", conn.RemoteAddr())

	//make a map and add the urls
	mapUrls := make(map[string]string)
	addUrl(mapUrls, "/", "home")

	//get the url and return the appropiate status
	getUrl(conn, mapUrls)
}
