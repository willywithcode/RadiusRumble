package clients

import (
	"log"
	"net/http"
	"server/internal/server"
	"server/internal/server/states"
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
	dbTx     *server.DbTx
	state    server.ClientStateHandler
}

func NewWebSocketClient(hub *server.Hub, writer http.ResponseWriter, request *http.Request) (server.ClientInterface, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return nil, err
	}
	var c = &WebSocketClient{
		id:       uint64(hub.Clients.Len()),
		conn:     conn,
		hub:      hub,
		logger:   log.Default(),
		sendChan: make(chan *packets.Packet, 256),
		dbTx:     hub.NewDbTx(),
	}
	return c, nil
}

// Implement all required methods for the ClientInterface
func (c *WebSocketClient) Id() uint64 {
	return c.id
}

func (c *WebSocketClient) ProcessMessage(senderId uint64, msg packets.Msg) {
	c.state.HandleMessage(senderId, msg)
}

func (c *WebSocketClient) Initialize(id uint64) {
	c.SetState(&states.Connected{})
}

func (c *WebSocketClient) SocketSend(msg packets.Msg) {
	c.SocketSendAs(c.id, msg)
}

func (c *WebSocketClient) DbTx() *server.DbTx {
	return c.dbTx
}

func (c *WebSocketClient) SharedGameObjects() *server.SharedGameObjects {
	return c.hub.SharedGameObject
}

func (c *WebSocketClient) SetState(state server.ClientStateHandler) {
	prevStateName := "None"
	if c.state != nil {
		prevStateName = c.state.Name()
		c.state.OnExit()
	}

	newStateName := "None"
	if state != nil {
		newStateName = state.Name()
	}

	c.logger.Printf("Switching from state %s to %s", prevStateName, newStateName)

	c.state = state

	if c.state != nil {
		c.state.SetClient(c)
		c.state.OnEnter()
	}
}

func (c *WebSocketClient) SocketSendAs(senderId uint64, msg packets.Msg) {
	select {
	case c.sendChan <- &packets.Packet{SenderId: senderId, Msg: msg}:
	default:
		c.logger.Println("Send channel is full, dropping message: ", msg)
	}
}

func (c *WebSocketClient) PassToPeer(msg packets.Msg, peerId uint64) {
	if peer, exists := c.hub.Clients.Get(peerId); exists {
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

	// Notify other players about this player leaving
	if c.state != nil && c.state.Name() == "Ingame" {
		c.Broadcast(packets.NewId(c.id)) // Use IdMessage to signal player disconnection
	}

	c.SetState(nil)
	c.hub.UnregisterChan <- c
	c.conn.Close()
	if _, closed := <-c.sendChan; !closed {
		close(c.sendChan)
	}
}
