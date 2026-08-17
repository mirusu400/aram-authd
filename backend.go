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
