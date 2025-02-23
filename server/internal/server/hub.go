package server

import (
	"log"
	"net/http"
	"server/pkg/packets"
)

type ClientInterface interface {
	Id() uint64
	ProcessMessage(senderId uint64, msg packets.Msg)
	Initialize(id uint64)
	SocketSend(msg packets.Msg)
	SocketSendAs(senderId uint64, msg packets.Msg)
	PassToPeer(msg packets.Msg, peerId uint64)
	Broadcast(msg packets.Msg)
	ReadPump()
	WritePump()
	Close(reason string)
}

type Hub struct {
	Clients        map[uint64]ClientInterface
	BroadcastChan  chan *packets.Packet
	RegisterChan   chan ClientInterface
	UnregisterChan chan ClientInterface
}

func NewHub() *Hub {
	return &Hub{
		Clients:        make(map[uint64]ClientInterface),
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
			client.Initialize(uint64(len(h.Clients)))
		case client := <-h.UnregisterChan:
			h.Clients[client.Id()] = nil
		case packet := <-h.BroadcastChan:
			for id, client := range h.Clients {
				if id != packet.SenderId {
					client.ProcessMessage(packet.SenderId, packet.Msg)
				}
			}
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
	h.RegisterChan <- client

	go client.WritePump()
	go client.ReadPump()
}
