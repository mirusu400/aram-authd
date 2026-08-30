package authd

import "testing"

// TestGrantGrantsAuthOrdinalsAndArmsCompletion pins that the bypass backend
// accepts the LGT DRM/auth ordinals and asks for the carrier response event,
// while leaving unrelated network traffic to the runtime default.
func TestGrantGrantsAuthOrdinalsAndArmsCompletion(t *testing.T) {
	var _ NetBackend = Grant{}
	var _ CompletionSource = Grant{}

	var grant Grant
	for _, ordinal := range []uint32{lgtAuthConnectOrdinal, lgtAuthStatusOrdinal} {
		result, handled := grant.Handle(Call{Ordinal: ordinal}, nil)
		if !handled || result != 0 {
			t.Fatalf("ordinal %d: result=%d handled=%v, want 0/true", ordinal, result, handled)
		}
		completion := grant.Complete(Call{Ordinal: ordinal})
		if completion == nil {
			t.Fatalf("ordinal %d: no completion armed", ordinal)
		}
		if completion.Event != lgtCarrierResponseEvent {
			t.Fatalf("ordinal %d: completion event = %d, want %d", ordinal, completion.Event, lgtCarrierResponseEvent)
		}
		if completion.DelayFrames <= 0 || len(completion.Response) == 0 {
			t.Fatalf("ordinal %d: completion = %+v, want a delayed non-empty response", ordinal, completion)
		}
	}

	// An unrelated ordinal is declined and arms nothing.
	if _, handled := grant.Handle(Call{Ordinal: 600}, nil); handled {
		t.Fatal("Grant handled an unrelated ordinal")
	}
	if completion := grant.Complete(Call{Ordinal: 600}); completion != nil {
		t.Fatalf("Grant armed a completion for an unrelated ordinal: %+v", completion)
	}
}
