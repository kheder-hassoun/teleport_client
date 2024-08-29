///   ***************************************  I hate Go lang ***************************************

package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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

///   ***************************************  configration ***************************************

//No need for config file this code will run in client side controlled by Descktop flutter application
//Don't worry it's my resposibilty

// ***************************************  main ***************************************
func main() {
	// you know whats this ?
	// this is a command-line flag parsing 😁
	//username and password defult value are just for testing
	var port = flag.String("p", "9999", "server port to use")
	var host = flag.String("h", "teleport.me", "server hostname to use")
	var username = flag.String("username", "test2", "username for authentication")
	var password = flag.String("password", "123456", "password for authentication")
	var sharedPort = flag.String("sharedport", "8000", "shared port to use")
	maxConnections := 100
	flag.Parse()

	if *username == "" || *password == "" {
		log.Fatal("Username and password must be provided")
	}
	//   ***************************************  Deffi hellman and authintaction section (dont say any thing 😥 it's take many hours to complite) ***************************************
	// this part ping pong with the server 🏓🏓
	curve := elliptic.P256()
	privKey, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	fatal(err)
	pubKey := elliptic.Marshal(curve, x, y)
	clientPubKeyHex := hex.EncodeToString(pubKey)

	conn, err := net.Dial("tcp", net.JoinHostPort(*host, *port))
	fatal(err)
	client := httputil.NewClientConn(conn, bufio.NewReader(conn))
	req, err := http.NewRequest("GET", "/", nil)
	fatal(err)

	req.Header.Set("X-Username", *username)
	req.Header.Set("X-Password", *password)
	req.Header.Set("X-Client-Public-Key", clientPubKeyHex)
	req.Host = net.JoinHostPort(*host, *port)
	log.Println("Sending request with username, password, and public key")
	client.Write(req)

	resp, _ := client.Read(req)

	serverPubKeyHex := resp.Header.Get("X-Server-Public-Key")
	if serverPubKeyHex == "" {
		log.Fatal("Server public key not received")
	}
	serverPubKey, err := hex.DecodeString(serverPubKeyHex)
	fatal(err)

	serverX, serverY := elliptic.Unmarshal(curve, serverPubKey)
	if serverX == nil || serverY == nil {
		log.Fatal("Failed to unmarshal server public key")
	}

	sharedX, _ := curve.ScalarMult(serverX, serverY, privKey)
	sharedSecret := sharedX.Bytes()

	//for my stupied testing
	//clientHash := sha256.Sum256(sharedSecret)
	//log.Printf("Client's shared secret hash: %x\n", clientHash[:])

	aesKey := sha256.Sum256(sharedSecret)
	block, err := aes.NewCipher(aesKey[:])
	fatal(err)
	aesGCM, err := cipher.NewGCM(block)
	fatal(err)

	fmt.Printf("HTTP service available at: http://%s\n", resp.Header.Get("X-Public-Host"))

	c, _ := client.Hijack()
	sess := session.New(c)
	defer sess.Close()

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConnections) // semaphore to limit connections for more more securty  a stupid user cannot hack me

	for {
		ch, err := sess.Accept()
		fatal(err)
		conn, err := net.Dial("tcp", "127.0.0.1:"+*sharedPort)
		fatal(err)
		sem <- struct{}{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release a slot
			handleSecureConnection(ch, conn, aesGCM)
		}()
	}
	wg.Wait()
}

//   ***************************************  handleConnection ***************************************

func handleSecureConnection(ch, conn io.ReadWriteCloser, aesGCM cipher.AEAD) {
	defer ch.Close()
	defer conn.Close()

	buffer := make([]byte, 4096)

	//here let's handle sending encrypted data from client to server .....
	go func() {
		defer conn.Close() // ensure that the connection is closed when done
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Println("Connection read error:", err)
				}
				return
			}

			nonce := make([]byte, aesGCM.NonceSize())
			if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
				log.Println("Error generating nonce:", err)
				return
			}
			log.Printf("Client nonce: %x\n", nonce)

			encryptedData := aesGCM.Seal(nil, nonce, buffer[:n], nil)
			log.Printf("Client encrypted data: %x\n", encryptedData)

			if _, err := ch.Write(nonce); err != nil {
				log.Println("Error writing nonce to channel:", err)
				return
			}
			if _, err := ch.Write(encryptedData); err != nil {
				log.Println("Error writing encrypted data to channel:", err)
				return
			}
		}
	}()

	// here we handle receiving encrypted data from server to client
	for {
		nonce := make([]byte, aesGCM.NonceSize())
		if _, err := io.ReadFull(ch, nonce); err != nil {
			if err != io.EOF {
				log.Println("Error reading nonce:", err)
			}
			return
		}
		log.Printf("Server nonce: %x\n", nonce)

		n, err := ch.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading encrypted data:", err)
			}
			return
		}
		log.Printf("Server received encrypted data: %x\n", buffer[:n])

		decryptedData, err := aesGCM.Open(nil, nonce, buffer[:n], nil)
		if err != nil {
			log.Println("Decryption error:", err)
			return
		}

		if _, err := conn.Write(decryptedData); err != nil {
			log.Println("Error writing decrypted data to connection:", err)
			return
		}
	}
}

//   *************************************** pretty  helper function ***************************************

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
