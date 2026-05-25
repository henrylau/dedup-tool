//go:build !server

package service

func (s *Similarity) IsServerMode() bool { return false }
