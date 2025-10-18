// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package bridge defines interfaces and implementations for bridging between the
// CLI and backend services. It provides abstractions for different transport
// mechanisms (gRPC, WebSocket, etc.) while maintaining a consistent interface
// for task distribution and event streaming during database seeding operations.
//
// The package enables pluggable transport implementations while providing a
// unified API for the CLI to interact with various backend services.
package bridge

import (
	"context"
	"seedfast/cli/internal/bridge/grpcclient"
	"seedfast/cli/internal/bridge/model"
	"seedfast/cli/internal/seeding"
)

// SQLTask models a unit of SQL work from backend.
// Deprecated: moved to model.SQLTask.
type SQLTask = model.SQLTask

// SQLResponse is the result of executing an SQLTask.
// Deprecated: moved to model.SQLResponse.
type SQLResponse = model.SQLResponse

// Bridge represents a connection to backend for transporting tasks and UI events.
type Bridge interface {
	// Connect establishes transport to backend. addr is gRPC address when using gRPC implementation.
	Connect(ctx context.Context, addr string, accessToken string) error
	// Init sends initial session parameters (sessionID may be empty to create new).
	Init(ctx context.Context, sessionID string, dbName string) error
	Close(ctx context.Context) error
	// Events returns a stream of seeding/logging events from backend for rendering.
	Events() <-chan seeding.Event
	// Tasks returns a stream of SQL tasks coming from backend.
	Tasks() <-chan model.SQLTask
	// SendSQLResponse sends result back to backend.
	SendSQLResponse(ctx context.Context, resp model.SQLResponse) error
}

// New creates a new bridge instance.
// It returns a gRPC client bridge.
func New() Bridge {
	return &grpcclient.Client{}
}
