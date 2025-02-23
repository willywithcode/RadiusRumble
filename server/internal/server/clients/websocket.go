package clients

import (
	"fmt"
	"log"
	"net/http"
	"server/internal/server"
	"server/pkg/packets"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type WebSocketClient struct {
	id       uint64
	conn     *websocket.Conn
	hub      *server.Hub
	logger   *log.Logger
	sendChan chan *packets.Packet
}

func NewWebSocketClient(hub *server.Hub, writer http.ResponseWriter, request *http.Request) (server.ClientInterface, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
	}
	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return nil, err
	}
	var c = &WebSocketClient{
		id:       uint64(len(hub.Clients)),
		conn:     conn,
		hub:      hub,
		logger:   log.Default(),
		sendChan: make(chan *packets.Packet),
	}
	return c, nil
}

// Implement all required methods for the ClientInterface
func (c *WebSocketClient) Id() uint64 {
	return c.id
}

func (c *WebSocketClient) ProcessMessage(senderId uint64, msg packets.Msg) {
}

func (c *WebSocketClient) Initialize(id uint64) {
	c.id = id
	c.logger.SetPrefix(fmt.Sprintf("Client %d: ", c.id))
}

func (c *WebSocketClient) SocketSend(msg packets.Msg) {
	c.SocketSendAs(c.id, msg)
}

func (c *WebSocketClient) SocketSendAs(senderId uint64, msg packets.Msg) {
	select {
	case c.sendChan <- &packets.Packet{SenderId: senderId, Msg: msg}:
	default:
		c.logger.Println("Send channel is full, dropping message: ", msg)
	}
}

func (c *WebSocketClient) PassToPeer(msg packets.Msg, peerId uint64) {
	if peer, exists := c.hub.Clients[peerId]; exists {
		peer.ProcessMessage(c.id, msg)
		return
	}
	c.logger.Println("Peer not found: ", peerId)
}

func (c *WebSocketClient) Broadcast(msg packets.Msg) {
	c.hub.BroadcastChan <- &packets.Packet{SenderId: c.id, Msg: msg}
}

func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.logger.Println("Closing read pump")
		c.Close("Read pump closed")
	}()
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Printf("error: %v", err)
			}
			break
		}
		packet := &packets.Packet{}
		err = proto.Unmarshal(data, packet)
		if err != nil {
			c.logger.Printf("error: %v", err)
			continue
		}
		if packet.SenderId == 0 {
			packet.SenderId = c.id
		}
		c.ProcessMessage(packet.SenderId, packet.Msg)
	}
}

func (c *WebSocketClient) WritePump() {
	defer func() {
		c.logger.Println("Closing write pump")
		c.Close("Write pump closed")
	}()
	for packet := range c.sendChan {
		writer, err := c.conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			c.logger.Printf("error: %v", err)
			return
		}
		data, err := proto.Marshal(packet)
		if err != nil {
			c.logger.Printf("error: %v", err)
			continue
		}
		_, err = writer.Write(data)
		if err != nil {
			c.logger.Printf("error: %v", err)
			continue
		}
		writer.Write([]byte{'\n'})
		err = writer.Close()
		if err != nil {
			c.logger.Printf("error: %v", err)
			continue
		}
	}
}

func (c *WebSocketClient) Close(reason string) {
	c.logger.Printf("Client %d disconnected: %s", c.id, reason)

	c.hub.UnregisterChan <- c
	c.conn.Close()
	if _, closed := <-c.sendChan; !closed {
		close(c.sendChan)
	}
}
