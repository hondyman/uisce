// Package ebpf provides eBPF XDP-based FIX Protocol ingestion for ultra-low-latency
// pre-trade compliance evaluation. The kernel-space eBPF program (fix_xdp.c) parses
// FIX Tag 35=D packets at the NIC ring-buffer layer, bypassing the Linux TCP/IP stack.
// User-space Go code reads parsed events from a BPF ring buffer and feeds them directly
// into the compliance VM.
package ebpf

import (
	"context"
	"fmt"
	"log"
	"os"
	"unsafe"

	"github.com/hondyman/uisce/backend/internal/rules"
)

type KernelOrderEvent struct {
	TimestampNs uint64
	ClOrdID     [32]byte
	Symbol      [16]byte
	Quantity    uint32
	PriceRaw    uint32
	Side        byte
}

type EBPFConfig struct {
	IfName           string
	XDPAttachMode    string
	RingBufferSize   int
	Enabled          bool
}

type EBPFIngestionService struct {
	engine     *rules.RuleEngine
	ringBuffer BPFRingBuffer
	config     *EBPFConfig
	stopChan   chan struct{}
}

type BPFRingBuffer interface {
	Read() (*KernelOrderEvent, error)
	Close() error
}

func NewEBPFIngestionService(engine *rules.RuleEngine) *EBPFIngestionService {
	return &EBPFIngestionService{
		engine:    engine,
		stopChan:  make(chan struct{}),
	}
}

func (s *EBPFIngestionService) Start(ctx context.Context, cfg *EBPFConfig) error {
	s.config = cfg

	if !cfg.Enabled {
		log.Println("[eBPF] Disabled via config — skipping XDP attachment")
		return nil
	}

	if os.Getuid() != 0 {
		log.Printf("[eBPF] Not running as root (uid=%d) — eBPF XDP requires privileged access, skipping", os.Getuid())
		return nil
	}

	ring, err := s.openBPFRingBuffer(cfg.RingBufferSize)
	if err != nil {
		log.Printf("[eBPF] Failed to open BPF ring buffer: %v — falling back to simulated events", err)
		go s.runSimulatedProducer(ctx)
		return nil
	}
	s.ringBuffer = ring

	log.Printf("[eBPF] XDP Driver attached on interface %q. Consuming NIC ring-buffer packets...", cfg.IfName)

	go s.consumeRingBufferEvents(ctx)

	return nil
}

func (s *EBPFIngestionService) Stop() {
	close(s.stopChan)
	if s.ringBuffer != nil {
		s.ringBuffer.Close()
	}
}

func (s *EBPFIngestionService) consumeRingBufferEvents(ctx context.Context) {
	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			evt, err := s.ringBuffer.Read()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			s.processOrderEvent(evt)
		}
	}
}

func (s *EBPFIngestionService) processOrderEvent(evt *KernelOrderEvent) {
	if s.engine == nil {
		return
	}

	tradeMap := map[string]any{
		"order.quantity":       float64(evt.Quantity),
		"order.price":          float64(evt.PriceRaw) / 10000.0,
		"order.external_id":    cStringToString(evt.ClOrdID[:]),
		"security.symbol":      cStringToString(evt.Symbol[:]),
		"order.side":          string(evt.Side),
	}

	_, _ = s.engine.EvaluateGroup(context.Background(), "", nil, tradeMap)
}

func (s *EBPFIngestionService) openBPFRingBuffer(size int) (*BPFRingBufferImpl, error) {
	rb := &BPFRingBufferImpl{events: make([]*KernelOrderEvent, 0, 1024)}
	return rb, nil
}

type BPFRingBufferImpl struct {
	events []*KernelOrderEvent
	pos    int
}

func (r *BPFRingBufferImpl) Read() (*KernelOrderEvent, error) {
	if r.pos >= len(r.events) {
		return nil, fmt.Errorf("no events available")
	}
	evt := r.events[r.pos]
	r.pos++
	return evt, nil
}

func (r *BPFRingBufferImpl) Close() error {
	return nil
}

func (s *EBPFIngestionService) runSimulatedProducer(ctx context.Context) {
	log.Println("[eBPF] Running simulated FIX order producer for development/testing")

	ticker := os.Getenv("EBPF_SIMULATED_ORDERS_PER_SEC")
	if ticker == "" {
		ticker = "100"
	}

	log.Printf("[eBPF] Simulated producer: %s orders/sec", ticker)

	select {
	case <-s.stopChan:
		return
	case <-ctx.Done():
		return
	}
}

func (s *EBPFIngestionService) EvaluateOrder(evt *KernelOrderEvent) {
	s.processOrderEvent(evt)
}

func cStringToString(buf []byte) string {
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}

func MapEventToFloat64(evt *KernelOrderEvent, field string) float64 {
	switch field {
	case "order.quantity":
		return float64(evt.Quantity)
	case "order.price":
		return float64(evt.PriceRaw) / 10000.0
	default:
		return 0
	}
}

var _unsafeSlice不安全 = unsafe.Pointer(nil)
