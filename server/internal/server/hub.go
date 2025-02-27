package server

import (
	"context"
	"database/sql"
	_ "embed"
	"log"
	"net/http"
	"server/internal/server/db"
	"server/internal/server/objects"
	"server/pkg/packets"

	_ "modernc.org/sqlite"
)

type ClientInterface interface {
	Id() uint64
	ProcessMessage(senderId uint64, msg packets.Msg)
	SetState(state ClientStateHandler)

	// A reference to the database transaction context
	DbTx() *DbTx

	Initialize(id uint64)
	SocketSend(msg packets.Msg)
	SocketSendAs(senderId uint64, msg packets.Msg)
	PassToPeer(msg packets.Msg, peerId uint64)
	Broadcast(msg packets.Msg)
	ReadPump()
	WritePump()
	Close(reason string)
}

type ClientStateHandler interface {
	Name() string
	SetClient(client ClientInterface)
	HandleMessage(senderId uint64, msg packets.Msg)
	OnExit()
	OnEnter()
}

type Hub struct {
	Clients        *objects.SharedCollection[ClientInterface]
	BroadcastChan  chan *packets.Packet
	RegisterChan   chan ClientInterface
	UnregisterChan chan ClientInterface
	dbPool         *sql.DB
}
type DbTx struct {
	Ctx     context.Context
	Queries *db.Queries
}

func (h *Hub) NewDbTx() *DbTx {
	return &DbTx{
		Ctx:     context.Background(),
		Queries: db.New(h.dbPool),
	}
}

var (
	//go:embed db/schema.sql
	schemaGenSql string
)

func NewHub() *Hub {
	dbPool, err := sql.Open("sqlite", "server.db")
	if err != nil {
		log.Fatal(err)
	}
	return &Hub{
		Clients:        objects.NewSharedCollection[ClientInterface](),
		BroadcastChan:  make(chan *packets.Packet, 256),
		RegisterChan:   make(chan ClientInterface, 256),
		UnregisterChan: make(chan ClientInterface, 256),
		dbPool:         dbPool,
	}
}

func (h *Hub) Run() {
	log.Println("Hub is running...")
	if _, err := h.dbPool.ExecContext(context.Background(), schemaGenSql); err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case client := <-h.RegisterChan:
			log.Printf("Registering new client with ID %d", h.Clients.Len())
			client.Initialize(h.Clients.Add(client))
		case client := <-h.UnregisterChan:
			log.Printf("Unregistering client %d", client.Id())
			h.Clients.Remove(client.Id())
		case packet := <-h.BroadcastChan:
			log.Printf("Broadcasting message from client %d", packet.SenderId)
			h.Clients.ForEach(func(id uint64, client ClientInterface) {
				if id != packet.SenderId {
					client.ProcessMessage(packet.SenderId, packet.Msg)
				}
			})
		}
	}
}

func (h *Hub) Serve(getNewClient func(*Hub, http.ResponseWriter, *http.Request) (ClientInterface, error), writer http.ResponseWriter, request *http.Request) {
	log.Printf("New connection attempt from %s", request.RemoteAddr)
	client, err := getNewClient(h, writer, request)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	log.Printf("Starting pumps for client %d", client.Id())
	go client.WritePump()
	go client.ReadPump()

	log.Printf("Registering client %d", client.Id())
	h.RegisterChan <- client
	log.Printf("Client %d registered successfully", client.Id())
}
