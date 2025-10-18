// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package grpcclient provides a gRPC-backed implementation of the Bridge interface.
// It implements the seeding bridge using gRPC protocol buffers for communication
// with the backend service. The client handles bidirectional streaming for both
// sending SQL tasks and receiving events, providing real-time progress updates
// during database seeding operations.
//
// The package manages connection lifecycle, stream handling, and protocol conversion
// between the internal model types and gRPC message formats.
package grpcclient

import (
    "context"
    "errors"
    "io"
    "crypto/tls"
    "time"
    "net"

    "seedfast/cli/internal/bridge/model"
    "seedfast/cli/internal/seeding"

    dbpb "seedfast/cli/internal/bridge/proto"

    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/credentials"
)

// Client implements bridge.Bridge using the DatabaseBridge.RunSeeding bidi stream.
type Client struct {
	conn   *grpc.ClientConn
	stream dbpb.DatabaseBridge_RunSeedingClient

	events chan seeding.Event
	tasks  chan model.SQLTask

	// In-memory token storage with TTL
	accessToken string
	tokenExpiry time.Time
}

// Connect dials the gRPC server and opens the RunSeeding stream.
// The access token is stored in-memory with a 20-minute TTL and sent with each gRPC request.
func (c *Client) Connect(ctx context.Context, addr string, accessToken string) error {
	// Store access token in-memory with 20-minute TTL
	c.accessToken = accessToken
	c.tokenExpiry = time.Now().Add(20 * time.Minute)

    // Derive SNI and ensure default port if missing
    host := addr
    if h, _, err := net.SplitHostPort(addr); err == nil {
        host = h
    }
    target := addr
    if _, _, err := net.SplitHostPort(addr); err != nil {
        target = net.JoinHostPort(addr, "443")
    }

    tlsCfg := &tls.Config{ ServerName: host, MinVersion: tls.VersionTLS12 }
    creds := credentials.NewTLS(tlsCfg)
    dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    var err error
    c.conn, err = grpc.DialContext(dctx, target, grpc.WithTransportCredentials(creds), grpc.WithBlock())
	if err != nil {
		return err
	}

	// Attach authorization metadata to context
	md := metadata.Pairs("authorization", "Bearer "+accessToken)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Python gRPC uses snake_case method names, so we need to use the literal proto name
	cs, sErr := c.conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}, "/database_bridge.DatabaseBridge/run_seeding")
	if sErr != nil {
		return sErr
	}
	c.stream = &grpc.GenericClientStream[dbpb.ClientMessage, dbpb.ServerMessage]{ClientStream: cs}
	c.events = make(chan seeding.Event, 64)
	c.tasks = make(chan model.SQLTask, 64)
	go func() { <-ctx.Done(); _ = c.Close(context.Background()) }()
	return nil
}

// Init sends initial session parameters and starts receiving.
func (c *Client) Init(ctx context.Context, sessionID string, dbName string) error {
	if c.stream == nil {
		return errors.New("stream not initialized")
	}
	if !c.isTokenValid() {
		return errors.New("access token expired or invalid")
	}
	if dbName == "" {
		return errors.New("dbName is required (cannot be empty)")
	}
	if err := c.stream.Send(&dbpb.ClientMessage{Message: &dbpb.ClientMessage_Init{Init: &dbpb.InitRequest{SessionId: sessionID, DbName: dbName}}}); err != nil {
		return err
	}
	go c.receiveLoop()
	return nil
}

func (c *Client) Close(ctx context.Context) error {
	// Clear access token from memory
	c.accessToken = ""
	c.tokenExpiry = time.Time{}

	if c.stream != nil {
		_ = c.stream.CloseSend()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) Events() <-chan seeding.Event { return c.events }
func (c *Client) Tasks() <-chan model.SQLTask  { return c.tasks }

// isTokenValid checks if the access token is still valid (not expired)
func (c *Client) isTokenValid() bool {
	return c.accessToken != "" && time.Now().Before(c.tokenExpiry)
}

// SendSQLResponse sends an SQL response to the server.
func (c *Client) SendSQLResponse(ctx context.Context, resp model.SQLResponse) error {
	if c.stream == nil {
		return errors.New("stream not initialized")
	}
	if !c.isTokenValid() {
		return errors.New("access token expired or invalid")
	}

	return c.stream.Send(&dbpb.ClientMessage{Message: &dbpb.ClientMessage_SqlResponse{SqlResponse: &dbpb.SQLResponse{
		RequestId:  resp.RequestID,
		Success:    resp.Success,
		ResultJson: resp.ResultJSON,
	}}})
}

func (c *Client) receiveLoop() {
	defer close(c.events)
	defer close(c.tasks)
	for {
		msg, err := c.stream.Recv()
		if err != nil {
			// Differentiate normal close vs error; avoid printing raw EOF as info in UI
			if errors.Is(err, io.EOF) {
				// Normal server close
				c.events <- seeding.Event{Type: seeding.EventType("stream_closed"), Message: "stream closed"}
			} else {
				if st, ok := status.FromError(err); ok {
					c.events <- seeding.Event{Type: seeding.EventType("stream_error"), Message: st.Code().String() + ": " + st.Message()}
				} else {
					c.events <- seeding.Event{Type: seeding.EventType("stream_error"), Message: err.Error()}
				}
			}
			return
		}
		switch m := msg.Message.(type) {
		case *dbpb.ServerMessage_SqlRequest:
			r := m.SqlRequest
			c.tasks <- model.SQLTask{RequestID: r.RequestId, SQLStatement: r.SqlStatement, IsWrite: r.IsWrite, Schema: r.Schema}
		case *dbpb.ServerMessage_UiEvent:
			u := m.UiEvent
			c.events <- seeding.Event{Type: seeding.EventType(u.EventType), Message: u.PayloadJson}
		}
	}
}
