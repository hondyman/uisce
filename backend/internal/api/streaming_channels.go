package api

import "sync"

// StreamChannel is an admin-defined streaming topic. The page designer's
// data-source picker offers these by name only — never a raw socket
// URL/topic string a client could supply — mirroring how the BO query path
// exposes boId/bindingId rather than a raw SQL string.
type StreamChannel string

const (
	// ChannelPrices carries market data (last price, bid/ask) keyed by
	// instrument ID. Instrument master data is not tenant-owned, so unlike
	// order-scoped channels this has no per-key ownership check yet.
	//
	// TODO(datasource-entitlement): gate ChannelPrices subscriptions on the
	// caller's security.Context.DatasourceID once instrument-level market
	// data entitlements exist. Until then this channel is allow-all for any
	// authenticated connection — deliberate for slice 1, not an oversight.
	ChannelPrices StreamChannel = "prices"
)

// knownChannels is the allow-list validated against on every subscribe.
var knownChannels = map[StreamChannel]bool{
	ChannelPrices: true,
}

// SubscriptionID identifies one subscribe call's worth of (channel, keys)
// for later unsubscribe. Returned to the client so it can manage many
// concurrent subscriptions unambiguously over one connection.
type SubscriptionID string

const (
	streamActionSubscribe   = "subscribe"
	streamActionUnsubscribe = "unsubscribe"
)

// streamActionRequest is the wire shape for both subscribe and unsubscribe.
// Identity is never carried on the wire — the server derives tenant/user
// from the connection's security.Context (set at ticket-authenticated
// upgrade), the same way validateScope trusts the JWT-derived tenant over
// any client-supplied value in the BO query path. Keys is plural from the
// start: a blotter grid subscribing per-instrument needs many keys in one
// call, not one message per key.
type streamActionRequest struct {
	Action         string         `json:"action"`
	Channel        StreamChannel  `json:"channel"`
	Keys           []string       `json:"keys"`
	SubscriptionID SubscriptionID `json:"subscriptionId,omitempty"` // required for unsubscribe
}

// streamActionResponse is returned for both subscribe and unsubscribe.
// Error is a single generic string on any failure — invalid channel,
// unknown key ownership, or anything else — so the subscribe path never
// becomes a cross-tenant existence oracle (same no-oracle reasoning as the
// ticket handler's uniform 401 body).
type streamActionResponse struct {
	Action         string         `json:"action"`
	SubscriptionID SubscriptionID `json:"subscriptionId,omitempty"`
	Channel        StreamChannel  `json:"channel,omitempty"`
	Keys           []string       `json:"keys,omitempty"`
	Error          string         `json:"error,omitempty"`
}

// subscription is one client's live listen on (channel, keys).
type subscription struct {
	id      SubscriptionID
	client  *WebSocketClient
	channel StreamChannel
	keys    map[string]struct{}
}

// channelKey is the reverse-index lookup key for fan-out: one entry per
// (channel, individual key), since a price update for "AAPL" must reach
// every subscription that listed AAPL among its keys, not just a
// subscription whose key set exactly matches.
type channelKey struct {
	channel StreamChannel
	key     string
}

var subIDCounter struct {
	mu sync.Mutex
	n  uint64
}

func nextSubscriptionID() SubscriptionID {
	subIDCounter.mu.Lock()
	defer subIDCounter.mu.Unlock()
	subIDCounter.n++
	return SubscriptionID(genSubIDPrefix + itoa(subIDCounter.n))
}

const genSubIDPrefix = "sub-"

// itoa avoids importing strconv solely for this in a tiny file; kept local
// and trivial on purpose.
func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// HandleSubscribe validates the request, performs the ownership check for
// the requested channel (currently a no-op for ChannelPrices — see the
// TODO(datasource-entitlement) above), and registers the subscription in
// both the client's own map and the hub's reverse index for fan-out.
func (h *WebSocketHub) HandleSubscribe(client *WebSocketClient, req streamActionRequest) streamActionResponse {
	if !knownChannels[req.Channel] {
		return streamActionResponse{Action: streamActionSubscribe, Error: "subscription denied"}
	}
	if len(req.Keys) == 0 {
		return streamActionResponse{Action: streamActionSubscribe, Error: "subscription denied"}
	}

	// Per-key ownership check goes here for tenant-owned channels
	// (order-updates, executions) in a later slice — e.g.
	// orderResolver.BelongsToTenant(key, client.tenantID) — returning the
	// same generic "subscription denied" on any failure, never
	// distinguishing "doesn't exist" from "not yours".

	sub := &subscription{
		id:      nextSubscriptionID(),
		client:  client,
		channel: req.Channel,
		keys:    make(map[string]struct{}, len(req.Keys)),
	}
	for _, k := range req.Keys {
		sub.keys[k] = struct{}{}
	}

	client.subsMu.Lock()
	if client.subscriptions == nil {
		client.subscriptions = make(map[SubscriptionID]*subscription)
	}
	client.subscriptions[sub.id] = sub
	client.subsMu.Unlock()

	h.subMu.Lock()
	for k := range sub.keys {
		ck := channelKey{channel: req.Channel, key: k}
		if h.subIndex[ck] == nil {
			h.subIndex[ck] = make(map[SubscriptionID]*subscription)
		}
		h.subIndex[ck][sub.id] = sub
	}
	h.subMu.Unlock()

	return streamActionResponse{
		Action:         streamActionSubscribe,
		SubscriptionID: sub.id,
		Channel:        req.Channel,
		Keys:           req.Keys,
	}
}

// HandleUnsubscribe removes one subscription (identified by SubscriptionID)
// from both the client's map and the hub's reverse index. Unsubscribing a
// subscription that isn't this client's own is rejected with the same
// generic error as any other denial.
func (h *WebSocketHub) HandleUnsubscribe(client *WebSocketClient, req streamActionRequest) streamActionResponse {
	client.subsMu.Lock()
	sub, ok := client.subscriptions[req.SubscriptionID]
	if ok {
		delete(client.subscriptions, req.SubscriptionID)
	}
	client.subsMu.Unlock()

	if !ok || sub.client != client {
		return streamActionResponse{Action: streamActionUnsubscribe, Error: "subscription denied"}
	}

	h.subMu.Lock()
	for k := range sub.keys {
		ck := channelKey{channel: sub.channel, key: k}
		delete(h.subIndex[ck], sub.id)
		if len(h.subIndex[ck]) == 0 {
			delete(h.subIndex, ck)
		}
	}
	h.subMu.Unlock()

	return streamActionResponse{Action: streamActionUnsubscribe, SubscriptionID: sub.id}
}

// teardownSubscriptions removes every subscription a disconnecting client
// held from the hub's reverse index. Called from run()'s unregister case so
// a dropped connection never leaves stale fan-out targets behind.
func (h *WebSocketHub) teardownSubscriptions(client *WebSocketClient) {
	client.subsMu.Lock()
	subs := client.subscriptions
	client.subscriptions = nil
	client.subsMu.Unlock()

	if len(subs) == 0 {
		return
	}

	h.subMu.Lock()
	defer h.subMu.Unlock()
	for _, sub := range subs {
		for k := range sub.keys {
			ck := channelKey{channel: sub.channel, key: k}
			delete(h.subIndex[ck], sub.id)
			if len(h.subIndex[ck]) == 0 {
				delete(h.subIndex, ck)
			}
		}
	}
}

// FanOutPriceUpdate delivers a price update for a single instrument key to
// every current subscription on ChannelPrices listening on that key.
// Delivery goes through client.send exactly like every other fan-out path
// in this hub (see sendJSON), preserving the single-writer-per-conn
// invariant, and a full send buffer drops this update rather than blocking
// — correct for market data, where a superseded tick is meant to be
// discarded in favor of the next one (latest-value-wins), which is also
// what the client-side ring buffer/quote table already assume.
func (h *WebSocketHub) FanOutPriceUpdate(key string, payload interface{}) {
	h.subMu.Lock()
	targets := h.subIndex[channelKey{channel: ChannelPrices, key: key}]
	// Copy under the lock; sendJSON below does its own I/O and must not run
	// while holding subMu.
	subs := make([]*subscription, 0, len(targets))
	for _, sub := range targets {
		subs = append(subs, sub)
	}
	h.subMu.Unlock()

	for _, sub := range subs {
		sub.client.sendJSON(struct {
			Type    string      `json:"type"`
			Channel string      `json:"channel"`
			Key     string      `json:"key"`
			Data    interface{} `json:"data"`
		}{Type: "tick", Channel: string(ChannelPrices), Key: key, Data: payload})
	}
}
