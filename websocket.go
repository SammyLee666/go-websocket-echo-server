package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"bytes"
	"log"
	"github.com/gorilla/websocket"
	"fmt"
	"time"
)

var addr = flag.String("addr", "0.0.0.0", "Input address")
var port = flag.String("port", ":8080", "Input port")
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const(
	PING_WAIT_TIME = 5 * time.Second
	PONG_WAIT_TIME = (PING_WAIT_TIME * 12) / 10
	WRITE_WAIT_TIME = 5 * time.Second
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

//等待接收消息
func (c *Client) Reader() {
	defer func() {
		c.conn.Close()
		close(c.send)
	}()

	c.conn.SetReadDeadline(time.Now().Add(PONG_WAIT_TIME))
	c.conn.SetPongHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(7*time.Second))
		return nil
	})

	for{
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("ReadMessage %s\n", err)
			return
		}
		c.send <- message
	}
}

//从channel拿到小消息发送
func (c *Client) Writer() {
	ticker := time.NewTicker(PING_WAIT_TIME)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for{
		select {
		case message := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(WRITE_WAIT_TIME))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("message sent %s\n", err)
				return
			}

		case <- ticker.C:
			//setting write deadline
			c.conn.SetWriteDeadline(time.Now().Add(WRITE_WAIT_TIME))
			//pong must be achieve after ping
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil{
				return
			}
		}

	}
}

func main(){
	flag.Parse()

	http.HandleFunc("/ws", WebSocketHandle)

	host := JoinString(*addr, *port)
	fmt.Printf("Listen Serve: %s\n", host)
	err := http.ListenAndServe(host, nil)
	if err != nil{
		log.Printf("ListenAndServe Error: %s\n", err)
	}

}

func WebSocketHandle(writer http.ResponseWriter, request *http.Request) {
	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Printf("Upgrade %s\n", err)
		return
	}

	client := &Client{conn:conn, send:make(chan []byte)}

	go client.Reader()
	go client.Writer()

}

func JoinString(args ...string) string {
	var buffer bytes.Buffer
	for _, str := range args{
		buffer.WriteString(str)
	}
	return buffer.String()
}