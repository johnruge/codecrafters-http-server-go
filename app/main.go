package main

import (
	"fmt"
	"io"
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
func getUrlAgentMethodBody (conn net.Conn) (string, string, string, string) {
	stringurl := getString(conn)

	//get the method
	var method string
	for i, v := range stringurl {
		if v == ' ' {
			method = stringurl[:i]
			break
		}
	}

	//the url and user-agent
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

	body := strings.TrimRight(parts[len(parts)-1], "\x00\r\n ")

	return url, userAgent, method, body
}

//this functyion handles getting files
func handleGetFiles(path string, conn net.Conn) {
	file, err := os.Open(path)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	stat, err := file.Stat()
	if err != nil {
		panic(err)
	}

	response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application" +
	"/octet-stream\r\nContent-Length: %d\r\n\r\n%s", stat.Size(), string(content))

	conn.Write([]byte(response))
}
// this function handles posting files
func handlePostFiles(path string, conn net.Conn, body string) {
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.WriteString(body)
	if err != nil {
		panic(err)
	}

	conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
}


// this implementation is for test cases where the unique path like {user_id} is at the end
// will implement a func that will work for all valid urls in later stages of the project
// currently this will not work home/{user_id}/{courses}/reviews
// this will work home/courses/reviews/{course_id}
// this also works for .../files/{file_name}
func getResponse (url string, userAgent string, method string, body string,
	mapUrls map[string]string, conn net.Conn) {
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
				switch val {
					case "unique":
						responseContent := url[i + 1:]
						response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: " +
						"text/plain\r\nContent-Length: %d\r\n\r\n%s", len(responseContent), responseContent)
						conn.Write([]byte(response))
						return
					case "file":
						path := fmt.Sprintf("/tmp/data/codecrafters.io/http-server-tester/%s", url[i+1:])
						if method == "GET" {
							handleGetFiles(path, conn)
							return
						} else {
							handlePostFiles(path, conn, body)
						}
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

// this function handles a connection
func handleconn(conn net.Conn) {
	fmt.Println("Accepted conection from: ", conn.RemoteAddr())
	for {
		//make a map and add the urls
		mapUrls := make(map[string]string)
		addUrl(mapUrls, "/", "static")
		addUrl(mapUrls, "/echo", "unique")
		addUrl(mapUrls, "/files", "file")

		//get the url and return the appropiate status
		url, userAgent, method, body := getUrlAgentMethodBody(conn)
		getResponse(url, userAgent, method, body, mapUrls, conn)
	}
}

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}
		go handleconn(conn)
	}
}
