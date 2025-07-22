package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

// represents an HTTP request
type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

// represents an HTTP response
type Response struct {
	StatusCode int
	StatusText string
	Headers    map[string]string
	Body       string
}

// creates a new Response with default values
func NewResponse() *Response {
	return &Response{
		Headers: make(map[string]string),
	}
}

// sets the status code and text for the response
func (r *Response) SetStatus(code int, text string) {
	r.StatusCode = code
	r.StatusText = text
}

// sends the response to the connection
func (r *Response) Write(conn net.Conn) error {
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n", r.StatusCode, r.StatusText)

	for key, value := range r.Headers {
		response += fmt.Sprintf("%s: %s\r\n", key, value)
	}

	if r.Body != "" {
		if _, exists := r.Headers["Content-Length"]; !exists {
			r.Headers["Content-Length"] = fmt.Sprintf("%d", len(r.Body))
			response += fmt.Sprintf("Content-Length: %d\r\n", len(r.Body))
		}
	}

	response += "\r\n" + r.Body

	_, err := conn.Write([]byte(response))
	return err
}

// parses the raw request string into a Request struct
func ParseRequest(rawRequest string) (*Request, error) {
	req := &Request{
		Headers: make(map[string]string),
	}

	parts := strings.Split(rawRequest, "\r\n")
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid request format")
	}

	// parse request line
	requestLine := strings.Split(parts[0], " ")
	if len(requestLine) < 3 {
		return nil, fmt.Errorf("invalid request line")
	}
	req.Method = requestLine[0]
	req.URL = requestLine[1]

	// parse headers
	var i int
	for i = 1; i < len(parts); i++ {
		if parts[i] == "" {
			break
		}
		headerParts := strings.SplitN(parts[i], ": ", 2)
		if len(headerParts) == 2 {
			req.Headers[headerParts[0]] = strings.TrimSpace(headerParts[1])
		}
	}

	// parse body
	if i < len(parts)-1 {
		req.Body = strings.TrimRight(strings.Join(parts[i+1:], "\r\n"), "\x00\r\n ")
	}

	return req, nil
}

func getString(conn net.Conn) (string, error) {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		if err == io.EOF {
			return "", err
		}
		return "", fmt.Errorf("failed to read from connection: %v", err)
	}
	return string(buffer[:n]), nil
}

func handleGetFiles(path string) *Response {
	resp := NewResponse()

	file, err := os.Open(path)
	if err != nil {
		resp.SetStatus(404, "Not Found")
		return resp
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		resp.SetStatus(500, "Internal Server Error")
		return resp
	}

	resp.SetStatus(200, "OK")
	resp.Headers["Content-Type"] = "application/octet-stream"
	resp.Body = string(content)
	return resp
}

func handlePostFiles(path string, body string) *Response {
	resp := NewResponse()

	file, err := os.Create(path)
	if err != nil {
		resp.SetStatus(500, "Internal Server Error")
		return resp
	}
	defer file.Close()

	_, err = file.WriteString(body)
	if err != nil {
		resp.SetStatus(500, "Internal Server Error")
		return resp
	}

	resp.SetStatus(201, "Created")
	return resp
}

func handleRequest(req *Request, mapUrls map[string]string) *Response {
	resp := NewResponse()

	if req.URL == "/user-agent" {
		userAgent := req.Headers["User-Agent"]
		resp.SetStatus(200, "OK")
		resp.Headers["Content-Type"] = "text/plain"
		resp.Body = userAgent
		return resp
	}

	for i, v := range req.URL {
		if i == len(req.URL)-1 {
			_, ok := mapUrls[req.URL]
			if ok {
				resp.SetStatus(200, "OK")
				return resp
			} else {
				resp.SetStatus(404, "Not Found")
				return resp
			}
		} else if v == '/' {
			if len(req.URL[:i]) == 0 {
				continue
			}
			val, ok := mapUrls[req.URL[:i]]
			if ok {
				switch val {
				case "unique":
					responseContent := req.URL[i+1:]
					resp.SetStatus(200, "OK")
					resp.Headers["Content-Type"] = "text/plain"
					resp.Body = responseContent
					return resp
				case "file":
					path := fmt.Sprintf("/tmp/data/codecrafters.io/http-server-tester/%s", req.URL[i+1:])
					if req.Method == "GET" {
						return handleGetFiles(path)
					} else {
						return handlePostFiles(path, req.Body)
					}
				}
			} else {
				resp.SetStatus(404, "Not Found")
				return resp
			}
		}
	}

	resp.SetStatus(404, "Not Found")
	return resp
}

func handleconn(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Accepted connection from:", conn.RemoteAddr())

	mapUrls := make(map[string]string)
	mapUrls["/"] = "static"
	mapUrls["/echo"] = "unique"
	mapUrls["/files"] = "file"

	for {
		rawRequest, err := getString(conn)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Printf("Error reading request: %v\n", err)
			resp := NewResponse()
			resp.SetStatus(400, "Bad Request")
			resp.Write(conn)
			return
		}

		req, err := ParseRequest(rawRequest)
		if err != nil {
			fmt.Printf("Error parsing request: %v\n", err)
			resp := NewResponse()
			resp.SetStatus(400, "Bad Request")
			resp.Write(conn)
			return
		}

		resp := handleRequest(req, mapUrls)

		// check if client requested connection close
		if connectionHeader, exists := req.Headers["Connection"]; exists && strings.ToLower(connectionHeader) == "close" {
			resp.Headers["Connection"] = "close"
		}

		if err := resp.Write(conn); err != nil {
			fmt.Printf("Error writing response: %v\n", err)
			return
		}

		// if Connection: close was requested, close the connection after response
		if connectionHeader, exists := resp.Headers["Connection"]; exists && strings.ToLower(connectionHeader) == "close" {
			return
		}
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
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}
		go handleconn(conn)
	}
}
