package balancer

import (
	"sync"
	"time"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
)

const Name = "node_breaker"

func init() {
	// Register a balancer builder that wraps the round_robin balancer.
	balancer.Register(newWrapperBuilder(Name, balancer.Get("round_robin")))
}

// --- 1. Wrapper Builder ---

// wrapperBuilder is a balancer builder that wraps another builder.
type wrapperBuilder struct {
	name    string
	builder balancer.Builder
}

func newWrapperBuilder(name string, builder balancer.Builder) balancer.Builder {
	return &wrapperBuilder{
		name:    name,
		builder: builder,
	}
}

func (b *wrapperBuilder) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	// Create our breakerBalancer first
	bb := &breakerBalancer{
		cc:       cc,
		breakers: make(map[string]*nodeCircuitBreaker),
		scToAddr: make(map[balancer.SubConn]string),
	}

	// Create a wrapped ClientConn to intercept picker updates
	wcc := &wrappedClientConn{
		ClientConn: cc,
		balancer:   bb,
	}

	// Build the underlying balancer with our wrapped ClientConn
	bb.Balancer = b.builder.Build(wcc, opts)

	return bb
}

func (b *wrapperBuilder) Name() string {
	return b.name
}

// --- 2. Breaker Balancer Wrapper ---

// breakerBalancer wraps a real balancer and intercepts its picker.
type breakerBalancer struct {
	balancer.Balancer // Embed the underlying balancer
	cc                balancer.ClientConn
	mu                sync.RWMutex
	breakers          map[string]*nodeCircuitBreaker // addr -> breaker
	scToAddr          map[balancer.SubConn]string
}

// UpdateClientConnState is called by gRPC when the resolver provides new addresses.
func (b *breakerBalancer) UpdateClientConnState(state balancer.ClientConnState) error {
	b.mu.Lock()
	// Create a map of the new addresses for easy lookup.
	newAddrs := make(map[string]struct{})
	for _, addr := range state.ResolverState.Addresses {
		newAddrs[addr.Addr] = struct{}{}
		if _, ok := b.breakers[addr.Addr]; !ok {
			// A new address appeared, create a circuit breaker for it.
			b.breakers[addr.Addr] = newNodeCircuitBreaker()
		}
	}

	// Clean up breakers for addresses that have been removed.
	for addr := range b.breakers {
		if _, ok := newAddrs[addr]; !ok {
			delete(b.breakers, addr)
		}
	}
	b.mu.Unlock()

	// Let the underlying balancer do its work
	return b.Balancer.UpdateClientConnState(balancer.ClientConnState{ResolverState: state.ResolverState, BalancerConfig: state.BalancerConfig})
}

// --- 3. Wrapped ClientConn to Intercept Picker Updates ---

type wrappedClientConn struct {
	balancer.ClientConn
	balancer *breakerBalancer
}

// UpdateState is called by the underlying balancer to update the picker.
// We intercept this call to wrap the picker.
func (w *wrappedClientConn) UpdateState(state balancer.State) {
	w.balancer.mu.Lock()
	defer w.balancer.mu.Unlock()

	// Wrap the picker from the underlying balancer with our breaker picker.
	newPicker := &breakerPicker{
		basePicker: state.Picker,
		balancer:   w.balancer,
	}

	// Update the parent ClientConn with the new state, but with our wrapped picker.
	w.ClientConn.UpdateState(balancer.State{ConnectivityState: state.ConnectivityState, Picker: newPicker})
}

// NewSubConn intercepts SubConn creation to maintain address mapping
func (w *wrappedClientConn) NewSubConn(addrs []resolver.Address, opts balancer.NewSubConnOptions) (balancer.SubConn, error) {
	sc, err := w.ClientConn.NewSubConn(addrs, opts)
	if err != nil {
		return nil, err
	}

	// Map SubConn to its first address (typically each SubConn has one address)
	if len(addrs) > 0 {
		w.balancer.mu.Lock()
		w.balancer.scToAddr[sc] = addrs[0].Addr
		w.balancer.mu.Unlock()
	}

	return sc, nil
}

// RemoveSubConn intercepts SubConn removal to clean up address mapping
func (w *wrappedClientConn) RemoveSubConn(sc balancer.SubConn) {
	w.balancer.mu.Lock()
	delete(w.balancer.scToAddr, sc)
	w.balancer.mu.Unlock()

	w.ClientConn.RemoveSubConn(sc)
}

// --- 4. Breaker Picker Wrapper ---

type breakerPicker struct {
	basePicker balancer.Picker
	balancer   *breakerBalancer
}

func (p *breakerPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	// Let the underlying picker (e.g., round_robin) choose a sub-connection.
	pickResult, err := p.basePicker.Pick(info)
	if err != nil {
		return pickResult, err
	}

	// Get the address of the chosen sub-connection.
	p.balancer.mu.RLock()
	addr, ok := p.balancer.scToAddr[pickResult.SubConn]
	breaker, breakerOk := p.balancer.breakers[addr]
	p.balancer.mu.RUnlock()

	if !ok || !breakerOk {
		// Should not happen, but as a fallback, just use the connection.
		return pickResult, nil
	}

	// Check the breaker state.
	if !breaker.Allow() {
		// Breaker is open, we should not use this connection.
		// We return an error to gRPC, which can trigger a retry if configured.
		return balancer.PickResult{}, status.Errorf(codes.Unavailable, "circuit breaker for node %s is open", addr)
	}

	// Breaker is not open, return the connection but wrap the Done function.
	return balancer.PickResult{
		SubConn: pickResult.SubConn,
		Done:    p.buildDoneFunc(breaker, pickResult.Done),
	}, nil
}

func (p *breakerPicker) buildDoneFunc(b *nodeCircuitBreaker, originalDone func(balancer.DoneInfo)) func(balancer.DoneInfo) {
	return func(info balancer.DoneInfo) {
		if isSystemError(info.Err) {
			b.RecordFailure()
		} else {
			b.RecordSuccess()
		}
		// Call the original Done function if it exists.
		if originalDone != nil {
			originalDone(info)
		}
	}
}

// --- 5. Self-Contained Circuit Breaker (Same as before) ---

type BreakerState int

const (
	StateClosed BreakerState = iota
	StateOpen
	StateHalfOpen
)

const (
	failureThreshold = 5
	successThreshold = 3
	openStateTimeout = 10 * time.Second
)

type nodeCircuitBreaker struct {
	mu                   sync.RWMutex
	state                BreakerState
	consecutiveFailures  int
	consecutiveSuccesses int
	lastFailureTime      time.Time
}

func newNodeCircuitBreaker() *nodeCircuitBreaker {
	return &nodeCircuitBreaker{state: StateClosed}
}

func (b *nodeCircuitBreaker) Allow() bool {
	b.mu.Lock() // Use write lock to handle state transition
	defer b.mu.Unlock()

	if b.state == StateOpen && time.Since(b.lastFailureTime) > openStateTimeout {
		b.state = StateHalfOpen
		b.consecutiveSuccesses = 0
	}
	return b.state != StateOpen
}

func (b *nodeCircuitBreaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == StateHalfOpen {
		b.consecutiveSuccesses++
		if b.consecutiveSuccesses >= successThreshold {
			b.state = StateClosed
			b.reset()
		}
	} else {
		b.reset()
	}
}

func (b *nodeCircuitBreaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFailures++
	if b.consecutiveFailures >= failureThreshold {
		b.state = StateOpen
		b.lastFailureTime = time.Now()
	}
}

func (b *nodeCircuitBreaker) reset() {
	b.consecutiveFailures = 0
	b.consecutiveSuccesses = 0
}

// --- 6. Helper Function (Same as before) ---

func isSystemError(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return true
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Internal, codes.ResourceExhausted, codes.DataLoss:
		return true
	default:
		return false
	}
}
