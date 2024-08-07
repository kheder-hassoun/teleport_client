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
	"sync"

	"github.com/progrium/qmux/golang/session"
)

var maxConnections int

func main() {
	var port = flag.String("p", "9999", "server port to use")
	var host = flag.String("h", "teleport.me", "server hostname to use")
	var subscription = flag.String("s", "free", "subscription level: free, moderate, high")
	flag.Parse()

	switch *subscription {
	case "free":
		maxConnections = 1
	case "moderate":
		maxConnections = 50
	case "high":
		maxConnections = 100
	default:
		log.Fatal("Unknown subscription level")
	}

	if flag.Arg(0) != "" {
		conn, err := net.Dial("tcp", net.JoinHostPort(*host, *port))
		fatal(err)
		client := httputil.NewClientConn(conn, bufio.NewReader(conn))
		req, err := http.NewRequest("GET", "/", nil)
		fatal(err)

		// Set the subscription level in the header
		req.Header.Set("X-Subscription-Level", *subscription)
		req.Host = net.JoinHostPort(*host, *port)
		log.Println("Sending request with subscription level:", *subscription)
		client.Write(req)
		resp, _ := client.Read(req)
		fmt.Printf("port %s http available at:\n", flag.Arg(0))
		fmt.Printf("http://%s\n", resp.Header.Get("X-Public-Host"))
		c, _ := client.Hijack()
		sess := session.New(c)
		defer sess.Close()

		var wg sync.WaitGroup
		sem := make(chan struct{}, maxConnections) // semaphore to limit connections

		for {
			ch, err := sess.Accept()
			fatal(err)
			conn, err := net.Dial("tcp", "127.0.0.1:"+flag.Arg(0))
			fatal(err)
			sem <- struct{}{} // acquire a slot
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }() // release a slot
				join(conn, ch)
			}()
		}
		wg.Wait()
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
