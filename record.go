package authd

import "sync"

// Event is one recorded network-ordinal call, captured during the
// reverse-engineering / observation phase.
type Event struct {
	Seq     int
	Ordinal uint32
	Args    [3]uint32
	Result  uint32
}

// Recorder is a NetBackend that logs every routed call and then delegates to an
// inner backend (Nop by default). It is the observation tool for RE'ing the
// request side of the handshake: run a game with a Recorder installed and read
// Events back to see exactly what the guest asks for.
type Recorder struct {
	Inner NetBackend

	mu     sync.Mutex
	events []Event
	seq    int
}

// NewRecorder returns a Recorder wrapping inner (Nop if nil).
func NewRecorder(inner NetBackend) *Recorder {
	if inner == nil {
		inner = Nop{}
	}
	return &Recorder{Inner: inner}
}

func (r *Recorder) Handle(call Call, mem Memory) (uint32, bool) {
	result, handled := r.Inner.Handle(call, mem)
	r.mu.Lock()
	r.seq++
	r.events = append(r.events, Event{
		Seq:     r.seq,
		Ordinal: call.Ordinal,
		Args:    call.Args,
		Result:  result,
	})
	r.mu.Unlock()
	return result, handled
}

// Events returns a copy of everything recorded so far.
func (r *Recorder) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Event, len(r.events))
	copy(out, r.events)
	return out
}
