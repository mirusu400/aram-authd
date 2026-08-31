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

func TestGrantHandlesAuthOrdinals(t *testing.T) {
	g := Grant{}
	for _, ord := range []uint32{106, 238} {
		if _, handled := g.Handle(Call{Ordinal: ord}, memStub{}); !handled {
			t.Fatalf("Grant must handle auth ordinal %d", ord)
		}
	}
	if _, handled := g.Handle(Call{Ordinal: 999}, memStub{}); handled {
		t.Fatal("Grant must decline non-auth ordinals")
	}
}

func TestGrantCompletionIsRequestDriven(t *testing.T) {
	g := Grant{}
	cases := []struct {
		name string
		call Call
		arg1 uint32
	}{
		// The opcode rides in r0 on the connect ordinal, r1 on the status one.
		{"connect ack", Call{Ordinal: 106, Args: [3]uint32{0xa600, 0, 0}}, 0},
		{"status keepalive", Call{Ordinal: 238, Args: [3]uint32{1, 0xa600, 0}}, 0},
		{"connect auth", Call{Ordinal: 106, Args: [3]uint32{0x5001, 4, 0}}, 0x5001},
		{"status auth", Call{Ordinal: 238, Args: [3]uint32{1, 0x5001, 4}}, 0x5001},
	}
	for _, tc := range cases {
		c := g.Complete(tc.call)
		if c == nil {
			t.Fatalf("%s: expected a completion", tc.name)
		}
		if c.Event != 1800 {
			t.Fatalf("%s: event %d, want 1800", tc.name, c.Event)
		}
		if c.Arg1 != tc.arg1 {
			t.Fatalf("%s: arg1 0x%x, want 0x%x", tc.name, c.Arg1, tc.arg1)
		}
	}
	if g.Complete(Call{Ordinal: 999}) != nil {
		t.Fatal("Grant must not complete a non-auth ordinal")
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
