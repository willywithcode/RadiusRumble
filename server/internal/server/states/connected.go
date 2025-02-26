package states

import (
	"fmt"
	"log"
	"server/internal/server"
	"server/pkg/packets"
)

type Connected struct {
	client server.ClientInterface
	logger *log.Logger
}

func (c *Connected) Name() string {
	return "Connected"
}

func (c *Connected) SetClient(client server.ClientInterface) {
	c.client = client
	var loggingPrefix = fmt.Sprintf("[%s] ", c.Name())
	c.logger = log.New(log.Writer(), loggingPrefix, log.LstdFlags)
}

func (c *Connected) OnEnter() {
	c.client.SocketSend(packets.NewId(c.client.Id()))
}
func (c *Connected) HandleMessage(senderId uint64, msg packets.Msg) {
}

func (c *Connected) OnExit() {
	c.logger.Println("OnExit")
}
