package states

import (
	"fmt"
	"log"
	"server/internal/server"
	"server/pkg/packets"
)

type Ingame struct {
	client server.ClientInterface
	logger *log.Logger
}

func (s *Ingame) Name() string {
	return "Ingame"
}

func (s *Ingame) SetClient(client server.ClientInterface) {
	s.client = client
	var loggingPrefix = fmt.Sprintf("[%s] ", s.Name())
	s.logger = log.New(log.Writer(), loggingPrefix, log.LstdFlags)
}

func (c *Ingame) HandleMessage(senderId uint64, msg packets.Msg) {
	if senderId == c.client.Id() {
		c.client.Broadcast(msg)
		return
	}
	c.client.SocketSendAs(senderId, msg)
}

func (s *Ingame) OnExit() {}

func (s *Ingame) OnEnter() {}
