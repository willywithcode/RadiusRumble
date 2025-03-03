package states

import (
	"context"
	"errors"
	"fmt"
	"log"
	"server/internal/server"
	"server/internal/server/db"
	"server/internal/server/objects"
	"server/pkg/packets"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Connected struct {
	client  server.ClientInterface
	logger  *log.Logger
	queries *db.Queries
	dbCtx   context.Context
}

func (c *Connected) Name() string {
	return "Connected"
}

func (c *Connected) SetClient(client server.ClientInterface) {
	c.client = client
	var loggingPrefix = fmt.Sprintf("[%s] ", c.Name())
	c.logger = log.New(log.Writer(), loggingPrefix, log.LstdFlags)
	c.queries = client.DbTx().Queries
	c.dbCtx = client.DbTx().Ctx
}

func (c *Connected) OnEnter() {
	c.client.SocketSend(packets.NewId(c.client.Id()))
}

func (c *Connected) HandleMessage(senderId uint64, msg packets.Msg) {
	switch msg := msg.(type) {
	case *packets.Packet_LoginRequest:
		c.handleLoginRequest(senderId, msg)
	case *packets.Packet_RegisterRequest:
		c.handleRegisterRequest(senderId, msg)
	}
}
func (c *Connected) OnExit() {
}
func (c *Connected) handleLoginRequest(senderId uint64, message *packets.Packet_LoginRequest) {
	if senderId != c.client.Id() {
		c.logger.Printf("Received login message from another client (Id %d)", senderId)
		return
	}

	username := message.LoginRequest.Username

	genericFailMessage := packets.NewDenyResponse("Incorrect username or password")

	user, err := c.queries.GetUserByUsername(c.dbCtx, strings.ToLower(username))
	if err != nil {
		c.logger.Printf("Error getting user %s: %v", username, err)
		c.client.SocketSend(genericFailMessage)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(message.LoginRequest.Password))
	if err != nil {
		c.logger.Printf("User entered wrong password: %s", username)
		c.client.SocketSend(genericFailMessage)
		return
	}

	c.logger.Printf("User %s logged in successfully", username)
	c.client.SocketSend(packets.NewOkResponse())
	c.client.SetState(&Ingame{
		player: &objects.Player{
			Name: username,
		},
	})
}
func (c *Connected) handleRegisterRequest(senderId uint64, message *packets.Packet_RegisterRequest) {
	if senderId != c.client.Id() {
		c.logger.Printf("Received register message from another client (Id %d)", senderId)
		return
	}

	username := strings.ToLower(message.RegisterRequest.Username)
	err := validateUsername(message.RegisterRequest.Username)
	if err != nil {
		reason := fmt.Sprintf("Invalid username: %v", err)
		c.logger.Println(reason)
		c.client.SocketSend(packets.NewDenyResponse(reason))
		return
	}

	_, err = c.queries.GetUserByUsername(c.dbCtx, username)
	if err == nil {
		c.logger.Printf("User already exists: %s", username)
		c.client.SocketSend(packets.NewDenyResponse("User already exists"))
		return
	}

	genericFailMessage := packets.NewDenyResponse("Error registering user (internal server error) - please try again later")

	// Add new user
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(message.RegisterRequest.Password), bcrypt.DefaultCost)
	if err != nil {
		c.logger.Printf("Failed to hash password: %s", username)
		c.client.SocketSend(genericFailMessage)
		return
	}

	_, err = c.queries.CreateUser(c.dbCtx, db.CreateUserParams{
		Username:     username,
		PasswordHash: string(passwordHash),
	})

	if err != nil {
		c.logger.Printf("Failed to create user %s: %v", username, err)
		c.client.SocketSend(genericFailMessage)
		return
	}

	c.client.SocketSend(packets.NewOkResponse())

	c.logger.Printf("User %s registered successfully", username)
}
func validateUsername(username string) error {
	if len(username) <= 0 {
		return errors.New("empty")
	}
	if len(username) > 20 {
		return errors.New("too long")
	}
	if username != strings.TrimSpace(username) {
		return errors.New("leading or trailing whitespace")
	}
	return nil
}
