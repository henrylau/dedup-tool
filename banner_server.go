//go:build server

package main

import "log"

func printStartupBanner() {
	log.Printf("folder-similarity: server listening on http://%s:%d", serverHost(), serverPort())
}
