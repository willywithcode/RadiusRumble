package states

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"server/internal/server"
	"server/internal/server/objects"
	"server/pkg/packets"
	"time"
)

type Ingame struct {
	client                 server.ClientInterface
	player                 *objects.Player
	logger                 *log.Logger
	cancelPlayerUpdateLoop context.CancelFunc
}

func (s *Ingame) Name() string {
	return "Ingame"
}

func (s *Ingame) SetClient(client server.ClientInterface) {
	s.client = client
	var loggingPrefix = fmt.Sprintf("[%s] ", s.Name())
	s.logger = log.New(log.Writer(), loggingPrefix, log.LstdFlags)
}

func (g *Ingame) HandleMessage(senderId uint64, msg packets.Msg) {
	switch message := msg.(type) {
	case *packets.Packet_Player:
		g.handlePlayer(senderId, message)
	case *packets.Packet_PlayerDirection:
		g.handlePlayerDirection(senderId, message)
	}
}
func (g *Ingame) handlePlayer(senderId uint64, message *packets.Packet_Player) {
	if senderId == g.client.Id() {
		g.logger.Println("Received player message from our own client, ignoring")
		return
	}
	g.client.SocketSendAs(senderId, message)
}

func (s *Ingame) OnExit() {
	if s.cancelPlayerUpdateLoop != nil {
		s.cancelPlayerUpdateLoop()
	}
	s.client.SharedGameObjects().Players.Remove(s.client.Id())
}

func (s *Ingame) OnEnter() {
	s.logger.Printf("Adding player %s to the shared collection", s.player.Name)
	go s.client.SharedGameObjects().Players.Add(s.player, s.client.Id())
	s.player.X = rand.Float64() * 100
	s.player.Y = rand.Float64() * 100
	s.player.Radius = 20
	s.player.Speed = 140

	// Send the initial player data to the client
	s.client.SocketSend(packets.NewPlayer(s.client.Id(), s.player))

	// Send information about all existing players to the new player
	s.client.SharedGameObjects().Players.ForEach(func(playerId uint64, player *objects.Player) {
		if playerId != s.client.Id() {
			s.logger.Printf("Sending existing player %s to new player", player.Name)
			s.client.SocketSendAs(playerId, packets.NewPlayer(playerId, player))
		}
	})
}
func (g *Ingame) syncPlayer(delta float64) {
	newX := g.player.X + g.player.Speed*math.Cos(g.player.Direction)*delta
	newY := g.player.Y + g.player.Speed*math.Sin(g.player.Direction)*delta

	g.player.X = newX
	g.player.Y = newY

	updatePacket := packets.NewPlayer(g.client.Id(), g.player)
	g.client.Broadcast(updatePacket)
	go g.client.SocketSend(updatePacket)
}
func (g *Ingame) playerUpdateLoop(ctx context.Context) {
	const delta float64 = 0.05
	ticker := time.NewTicker(time.Duration(delta*1000) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.syncPlayer(delta)
		case <-ctx.Done():
			return
		}
	}
}
func (g *Ingame) handlePlayerDirection(senderId uint64, message *packets.Packet_PlayerDirection) {
	if senderId == g.client.Id() {
		g.player.Direction = message.PlayerDirection.Direction

		// If this is the first time receiving a player direction message from our client, start the player update loop
		if g.cancelPlayerUpdateLoop == nil {
			ctx, cancel := context.WithCancel(context.Background())
			g.cancelPlayerUpdateLoop = cancel
			go g.playerUpdateLoop(ctx)
		}
	}
}
