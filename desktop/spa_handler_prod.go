//go:build !verify

package main

func initVerifyHandler(h *SPAHandler) {
	// In production builds, zero debug or test verification endpoints are mounted.
}
