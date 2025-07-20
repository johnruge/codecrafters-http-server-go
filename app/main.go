package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

var _ = net.Listen
var _ = os.Exit

func getUrl (conn net.Conn) string {
	//make a buffer and read the request
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Failed to read the buffer")
		os.Exit(1)
	}

	//change the buffer to a string, get the url
	stringurl := string(buffer)
	parts := strings.Split(stringurl, "\r\n")
	requestParts := (strings.Split(parts[0], " "))
	if len(requestParts) < 2 {
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		os.Exit(1)
	}

	return requestParts[1]
}

// this implementation is for test cases where the unique path like {user_id} is at the end
// i will implement the function that will work for all test cases and valid urls in later stages of the project
// currently this will not work home/{user_id}/{courses}/reviews
// this will work home/courses/reviews/{course_id}
func getResponse (url string, mapUrls map[string]string, conn net.Conn) {
	for i, v := range url {
		if  i == len(url) - 1 {
			_, ok := mapUrls[url]
			if ok {
				conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
				return
			} else {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
				return
			}
		} else if v == '/' {
			if len(url[:i]) == 0 {
				//pass
			} else {
			val, ok := mapUrls[url[:i]]
			if ok {
				if val == "unique" {
					responseContent := url[i + 1:]
					response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(responseContent), responseContent)
					conn.Write([]byte(response))
					return
				}
			} else {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
				return
			}}
		}
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
	addUrl(mapUrls, "/", "static")
	addUrl(mapUrls, "/echo", "unique")

	//get the url and return the appropiate status
	url := getUrl(conn)
	fmt.Println(url)
	getResponse(url, mapUrls, conn)

}
