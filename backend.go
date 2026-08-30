// Package authd emulates the LGT carrier DRM / server-authentication handshake
// that LGT WIPI-C (raptor) games poll for before they will start.
//
// aram-core routes the raptor network ordinals (106/238 and any related
// connect/send/recv) to a NetBackend. The default is a no-op that preserves
// today's behavior; aram-emu injects a real backend from this package.
package authd

// Memory is the guest-memory view a backend uses to read the game's requests
// and populate the session state the game reads back. All addresses are guest
// addresses; errors mirror the underlying CPU backend.
type Memory interface {
	ReadU8(addr uint32) (uint8, error)
	WriteU8(addr uint32, value uint8) error
	ReadU32(addr uint32) (uint32, error)
	WriteU32(addr uint32, value uint32) error
	ReadBytes(addr uint32, n int) ([]byte, error)
	WriteBytes(addr uint32, data []byte) error
}

// Call is one raptor network-ordinal invocation. Args holds r0..r2 as the guest
// passed them.
type Call struct {
	Ordinal uint32
	Args    [3]uint32
}

// NetBackend models the LGT carrier network/DRM service. aram-core calls Handle
// for each routed ordinal. A backend may read/write guest session memory via
// mem and returns the guest-visible result (r0). handled=false means the
// backend declines the ordinal and the runtime should apply its default.
type NetBackend interface {
	Handle(call Call, mem Memory) (result uint32, handled bool)
}

// Nop is the default backend: it declines every ordinal, preserving the
// runtime's existing (unimplemented) behavior.
type Nop struct{}

func (Nop) Handle(Call, Memory) (uint32, bool) { return 0, false }

// Completion is the asynchronous carrier response a backend asks the runtime to
// deliver to the title after a network ordinal. A real handset receives the
// DRM/server handshake result out of band; a raptor title blocks on
// "접속중"/"서버 접속중" until it arrives. The runtime posts it as a clet event
// HandleEvent(Event, Arg1, buffer) with Response written to the buffer, after
// DelayFrames so the title has settled into its wait state.
type Completion struct {
	Event       uint32
	Arg1        uint32
	Response    []byte
	DelayFrames int
}

// CompletionSource is an optional NetBackend capability: after the runtime
// routes a handled network ordinal, it asks for an asynchronous completion to
// deliver, or nil for none.
type CompletionSource interface {
	Complete(call Call) *Completion
}

const (
	// The LGT carrier DRM/auth handshake a raptor Clet drives through these
	// ordinals; the runtime routes them here.
	lgtAuthConnectOrdinal = uint32(106)
	lgtAuthStatusOrdinal  = uint32(238)

	// lgtCarrierResponseEvent is the CletHandleEvent type LGT delivers the
	// server response on. 하이브리드 releases its "서버 접속중" wait when it
	// arrives; the handler reads a small response struct from the event's data
	// pointer and its status from arg1, so a zeroed success response suffices.
	lgtCarrierResponseEvent = uint32(1800)

	// lgtCompletionDelayFrames lets the title finish registering its session
	// and paint its wait screen before the response is posted.
	lgtCompletionDelayFrames = 3
)

// Grant is a NetBackend that emulates a successful LGT carrier DRM/auth
// handshake: it accepts the auth ordinals and delivers a synthetic success
// response, so titles that gate startup on the carrier server run without a
// live carrier (the "인증 우회" cheat). It handles only the auth ordinals so
// unrelated network traffic still falls through to the runtime default.
type Grant struct{}

func (Grant) Handle(call Call, _ Memory) (uint32, bool) {
	switch call.Ordinal {
	case lgtAuthConnectOrdinal, lgtAuthStatusOrdinal:
		return 0, true
	}
	return 0, false
}

func (Grant) Complete(call Call) *Completion {
	switch call.Ordinal {
	case lgtAuthConnectOrdinal, lgtAuthStatusOrdinal:
		return &Completion{
			Event:       lgtCarrierResponseEvent,
			Arg1:        0,
			Response:    make([]byte, 16),
			DelayFrames: lgtCompletionDelayFrames,
		}
	}
	return nil
}
