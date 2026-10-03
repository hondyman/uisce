package infra

import (
	"crypto/rand"
	"io"
)

func cryptoRand() io.Reader { return rand.Reader }
