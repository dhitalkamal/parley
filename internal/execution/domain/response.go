package execution

import (
	"time"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

// Response is what came back from executing a collection.Request.
type Response struct {
	StatusCode int
	Status     string
	Headers    []collection.Header
	Body       []byte
	Elapsed    time.Duration
	Timing     Timing
}

// Timing breaks Elapsed down into the phases of the underlying HTTP
// round trip, captured via net/http/httptrace - backs the Response
// viewer's Timeline tab. Any phase that didn't happen (e.g. DNSLookup for
// a literal IP, TLSHandshake for plain HTTP, or a connection reused from
// the pool) is left at zero rather than estimated.
type Timing struct {
	DNSLookup       time.Duration
	TCPConnect      time.Duration
	TLSHandshake    time.Duration
	ServerWait      time.Duration // request fully written to first response byte
	ContentTransfer time.Duration // first response byte to the body fully read
}
