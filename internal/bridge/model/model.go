// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package model defines shared data structures for bridge communication.
// It provides type definitions for SQL tasks, responses, and other data
// structures that are exchanged between the CLI and backend services
// through various bridge implementations.
//
// The types in this package are designed to be transport-agnostic and
// provide a stable interface for different communication protocols.
package model

// SQLTask models a unit of SQL work from backend.
type SQLTask struct {
	RequestID    string
	SessionID    string
	SQLStatement string
	IsWrite      bool
	Schema       string // Database schema to use for the query
}

// SQLResponse is the result of executing an SQLTask.
type SQLResponse struct {
	RequestID  string
	Success    bool
	ResultJSON string
}
