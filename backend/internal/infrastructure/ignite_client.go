package infrastructure

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	ignite "github.com/amsokol/ignite-go-client/binary/v1"
)

type IgniteClient struct {
	Client ignite.Client
}

func NewIgniteClient(addr string) (*IgniteClient, error) {
	if addr == "" {
		return nil, fmt.Errorf("ignite address is empty")
	}

	host := addr
	port := 10800 // default

	// Use net.SplitHostPort if it contains a colon
	if strings.Contains(addr, ":") {
		h, p, err := net.SplitHostPort(addr)
		if err == nil {
			host = h
			pi, err := strconv.Atoi(p)
			if err == nil {
				port = pi
			}
		}
	}

	c, err := ignite.Connect(ignite.ConnInfo{
		Network: "tcp",
		Host:    host,
		Port:    port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ignite: %w", err)
	}

	return &IgniteClient{Client: c}, nil
}

func (c *IgniteClient) Put(cacheName string, key interface{}, value interface{}) error {
	// v1 signature: CacheGetOrCreateWithName(name string)
	// Create cache if not exists
	err := c.Client.CacheGetOrCreateWithName(cacheName)
	if err != nil {
		return fmt.Errorf("failed to get/create cache: %w", err)
	}

	// v1 signature: CachePut(cache string, binary bool, key interface{}, val interface{})
	// We set binary=false assuming primitive/simple types or strictly typed objects.
	// Put
	err = c.Client.CachePut(cacheName, false, key, value)
	return err
}

func (c *IgniteClient) Get(cacheName string, key interface{}) (interface{}, error) {
	// v1 signature: CacheGet(cache string, binary bool, key interface{})
	return c.Client.CacheGet(cacheName, false, key)
}

func (c *IgniteClient) Close() error {
	return c.Client.Close()
}
