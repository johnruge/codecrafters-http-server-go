package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

var _ = net.Listen
var _ = os.Exit

//this func reads bytes from a connection and returns string
func getString (conn net.Conn) string {
	//make a buffer and read the request
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Failed to read the buffer")
		os.Exit(1)
	}

	stringurl := string(buffer)
	return stringurl
}

//this func gets the url and user-agent from a request
func getUrlAgent (conn net.Conn) (string, string) {
	stringurl := getString(conn)
	parts := strings.Split(stringurl, "\r\n")
	if len(parts) < 4 {
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		os.Exit(1)
	}

	url := (strings.Split(parts[0], " "))[1]
	var userAgent string

	// Retrive the User-agent from the header
	for _, v := range parts[1:] {
		key := (strings.Split(v, " "))[0]
		if key == "User-Agent:" {
			userAgent = (strings.Split(v, " "))[1]
			break
		}
	}

	return url, userAgent
}


// this implementation is for test cases where the unique path like {user_id} is at the end
// will implement a func that will work for all valid urls in later stages of the project
// currently this will not work home/{user_id}/{courses}/reviews
// this will work home/courses/reviews/{course_id}
func getResponse (url string, userAgent string, mapUrls map[string]string, conn net.Conn) {
	if url == "/user-agent" {
		response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: " +
		"text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)
		conn.Write([]byte(response))
		return
	}

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
					response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: " +
					"text/plain\r\nContent-Length: %d\r\n\r\n%s", len(responseContent), responseContent)
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
	url, userAgent := getUrlAgent(conn)
	getResponse(url, userAgent, mapUrls, conn)
}
