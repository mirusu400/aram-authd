package authd

import "testing"

// memStub is a tiny in-memory Memory for tests.
type memStub map[uint32]byte

func (m memStub) ReadU8(a uint32) (uint8, error)  { return m[a], nil }
func (m memStub) WriteU8(a uint32, v uint8) error { m[a] = v; return nil }
func (m memStub) ReadU32(a uint32) (uint32, error) {
	return uint32(m[a]) | uint32(m[a+1])<<8 | uint32(m[a+2])<<16 | uint32(m[a+3])<<24, nil
}
func (m memStub) WriteU32(a uint32, v uint32) error {
	m[a], m[a+1], m[a+2], m[a+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	return nil
}
func (m memStub) ReadBytes(a uint32, n int) ([]byte, error) {
	b := make([]byte, n)
	for i := range b {
		b[i] = m[a+uint32(i)]
	}
	return b, nil
}
func (m memStub) WriteBytes(a uint32, d []byte) error {
	for i, v := range d {
		m[a+uint32(i)] = v
	}
	return nil
}

func TestNopDeclines(t *testing.T) {
	if _, handled := (Nop{}).Handle(Call{Ordinal: 106}, memStub{}); handled {
		t.Fatal("Nop must decline every ordinal")
	}
}

func TestRecorderLogsAndDelegates(t *testing.T) {
	r := NewRecorder(nil)
	r.Handle(Call{Ordinal: 106, Args: [3]uint32{0xa600, 0, 0}}, memStub{})
	r.Handle(Call{Ordinal: 238, Args: [3]uint32{0, 0xa600, 0}}, memStub{})
	ev := r.Events()
	if len(ev) != 2 {
		t.Fatalf("recorded %d events, want 2", len(ev))
	}
	if ev[0].Ordinal != 106 || ev[0].Args[0] != 0xa600 || ev[0].Seq != 1 {
		t.Fatalf("unexpected first event: %+v", ev[0])
	}
	if ev[1].Ordinal != 238 || ev[1].Seq != 2 {
		t.Fatalf("unexpected second event: %+v", ev[1])
	}
}
