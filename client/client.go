package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"

	//? for handeling virtual host  actually in http request
	"github.com/progrium/qmux/golang/session"
)

func main() {
	var port = flag.String("p", "9999", "server port to use") //* here we define a command line flage name it p it's specific the server port and defult value is 9999
	var host = flag.String("h", "teleport.me", "server hostname to use")
	flag.Parse() //? now we can get value by name ('port','host','addr')

	//---------------------------------------------------------------------------------------

	//! client usage: teleport [-h=<server hostname>] <local port>

	if flag.Arg(0) != "" { //flag.Arg(0) returns the first non-flag argument
		//* establishes a TCP connection to the server specified by host and port.
		conn, err := net.Dial("tcp", net.JoinHostPort(*host, *port))
		// conn, err := net.Dial("tcp", net.JoinHostPort("192.168.91.131", *port)) //for linux
		fatal(err) //? checks if an error occurred and logs it, then terminates the program

		//*creates a new HTTP client connection //bufio.NewReader(conn)//wraps the connection in a buffered reader to improve efficiency.
		client := httputil.NewClientConn(conn, bufio.NewReader(conn))
		req, err := http.NewRequest("GET", "/", nil) //*  create a new HTTP GET request targeting the root path ("/").

		//? set the Host header to the server's host and port then  join port and host like "vcap.me:9999".
		req.Host = net.JoinHostPort(*host, *port)
		fatal(err)

		client.Write(req)           //send the HTTP request to the server.
		resp, _ := client.Read(req) // read the respose from server
		fmt.Printf("port %s http available at:\n", flag.Arg(0))
		fmt.Printf("http://%s\n", resp.Header.Get("X-Public-Host")) //! read url from header response

		//? Hijacking an HTTP connection means taking over the underlying network connection from the HTTP client.
		//? this allows  to use the connection for non-HTTP purposes.

		c, _ := client.Hijack()
		sess := session.New(c)
		//? defer 😍 its like close it but not now wait untel the end // close the session after the function return
		defer sess.Close()

		//*this loop continuously accepts new channels from the session.
		//*for each accepted channel, it establishes a new TCP connection to the local server at the provided port.

		for {
			ch, err := sess.Accept()
			fatal(err)
			conn, err := net.Dial("tcp", "127.0.0.1:"+flag.Arg(0))
			fatal(err)
			go join(conn, ch) //? starts a ❤ goroutine ❤ to link the channel to the local server connection using the join function.
		}
		return
	}

}
func join(a io.ReadWriteCloser, b io.ReadWriteCloser) {
	go io.Copy(b, a)
	io.Copy(a, b)
	a.Close()
	b.Close()
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
