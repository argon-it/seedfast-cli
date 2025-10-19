// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

//go:build !darwin

package keychain

import "errors"

// securityBackend is a stub for non-macOS platforms.
type securityBackend struct{}

// newSecurityBackend returns an error on non-macOS platforms.
func newSecurityBackend() (*securityBackend, error) {
	return nil, errors.New("security backend only available on macOS")
}

func (s *securityBackend) Set(key, value string) error {
	return errors.New("not implemented")
}

func (s *securityBackend) Get(key string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *securityBackend) Delete(key string) error {
	return errors.New("not implemented")
}
