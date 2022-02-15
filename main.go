package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

var (
	listenFlag string
	connFlag   string
)

func main() {
	flag.StringVar(&listenFlag, "listen", "", "Server listening address")
	flag.StringVar(&connFlag, "connect", "", "Address to connect to")
	flag.Parse()

	if listenFlag != "" && connFlag != "" {
		panic("only one of server or connection address can be specified")
	}

	reader := bufio.NewReader(os.Stdin)

	switch {
	case connFlag != "":
		fmt.Print("Nickname: ")
		input, _ := reader.ReadString('\n')
		nick := strings.TrimSpace(input)

		client := NewClient(connFlag, nick)
		for {
			msg, _ := reader.ReadString('\n')
			client.Send(strings.TrimSpace(msg))
		}
	default:
		if listenFlag == "" {
			listenFlag = "localhost:9999"
		}

		server := NewServer(listenFlag)
		for {
			msg, _ := reader.ReadString('\n')
			server.Send(strings.TrimSpace(msg))
		}
	}
}

type Client struct {
	nick string
	sock net.Conn
}

func NewClient(addr, nick string) *Client {
	sock, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			msg := make([]byte, 4096)
			if _, err := sock.Read(msg); err != nil {
				panic(err)
			}

			fmt.Println(string(msg))
		}
	}()

	return &Client{nick, sock}

}

func (c *Client) Send(msg string) {
	_, _ = c.sock.Write([]byte(fmt.Sprintf("%s: %s", c.nick, msg)))
}

type Server struct {
	socks     []net.Conn
	broadcast chan Broadcast
}

type Broadcast struct {
	msg    []byte
	remote string
}

func NewServer(addr string) *Server {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	socks := make([]net.Conn, 0)
	broadcast := make(chan Broadcast)

	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil {
				continue
			}

			remote := conn.RemoteAddr().String()

			fmt.Printf("new connection: %s\n", remote)
			socks = append(socks, conn)
			go func(c net.Conn) {
				for {
					msg := make([]byte, 4096)
					if _, err := c.Read(msg); err != nil {
						fmt.Printf("closed: %s\n", remote)
						break
					}

					fmt.Println(string(msg))
					broadcast <- Broadcast{msg, remote}
				}

				_ = c.Close()
			}(conn)
		}
	}()

	go func() {
		for {
			packet := <-broadcast
			for _, sock := range socks {
				if packet.remote == sock.RemoteAddr().String() {
					continue
				}

				_, _ = sock.Write(packet.msg)
			}
		}
	}()

	return &Server{socks, broadcast}
}

func (s *Server) Send(msg string) {
	s.broadcast <- Broadcast{
		msg: []byte(fmt.Sprintf("[%s]", msg)),
		remote: "srv",
	}
}
