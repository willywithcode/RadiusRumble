package server

import (
	"log"
	"net/http"
	"server/internal/server/objects"
	"server/pkg/packets"
)

type ClientInterface interface {
	Id() uint64
	ProcessMessage(senderId uint64, msg packets.Msg)
	SetState(state ClientStateHandler)
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
}

type Hub struct {
	Clients        *objects.SharedCollection[ClientInterface]
	BroadcastChan  chan *packets.Packet
	RegisterChan   chan ClientInterface
	UnregisterChan chan ClientInterface
}

func NewHub() *Hub {
	return &Hub{
		Clients:        objects.NewSharedCollection[ClientInterface](),
		BroadcastChan:  make(chan *packets.Packet),
		RegisterChan:   make(chan ClientInterface),
		UnregisterChan: make(chan ClientInterface),
	}
}

func (h *Hub) Run() {
	log.Println("Awaiting clients registration...")

	for {
		select {
		case client := <-h.RegisterChan:
			client.Initialize(h.Clients.Add(client))
		case client := <-h.UnregisterChan:
			h.Clients.Remove(client.Id())
		case packet := <-h.BroadcastChan:
			h.Clients.ForEach(func(id uint64, client ClientInterface) {
				if id != packet.SenderId {
					client.ProcessMessage(packet.SenderId, packet.Msg)
				}
			})
		}
	}
}

func (h *Hub) Serve(getNewClient func(*Hub, http.ResponseWriter, *http.Request) (ClientInterface, error), writer http.ResponseWriter, request *http.Request) {
	log.Println("New Client Connected from ", request.RemoteAddr)
	client, err := getNewClient(h, writer, request)
	if err != nil {
		log.Println("Error getting new client:", err)
		return
	}

	// Start pumps before registering to ensure channel is ready
	go client.WritePump()
	go client.ReadPump()

	// Register after pumps are started
	h.RegisterChan <- client
}
