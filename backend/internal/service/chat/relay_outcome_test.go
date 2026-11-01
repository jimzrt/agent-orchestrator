package chat_test

import (
	"context"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// `ao send` reports the outcome the relay already knows: an idle session starts
// a turn, a busy one queues the message with its place in the queue.
func TestRelaySessionChatTurnReportsStartedAndQueuePosition(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	idle, err := h.svc.RelaySessionChatTurn(ctx, testSession, "first", "", ports.MessageDeliveryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if idle.Kind != ports.MessageDeliveryStarted || idle.TurnID == "" || idle.QueuePosition != 0 {
		t.Fatalf("idle relay = %+v, want a started turn with no queue position", idle)
	}
	h.conv.emit(ports.ChatEvent{Kind: ports.ChatEventTurnStarted, ProviderTurnID: "provider-turn-1"})

	for want, text := range []string{"second", "third"} {
		busy, err := h.svc.RelaySessionChatTurn(ctx, testSession, text, "", ports.MessageDeliveryOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if busy.Kind != ports.MessageDeliveryQueued || busy.TurnID == "" || busy.QueuePosition != want+1 {
			t.Fatalf("busy relay %q = %+v, want queued at position %d", text, busy, want+1)
		}
	}
	if got := len(h.conv.sentTexts()); got != 1 {
		t.Fatalf("provider calls = %d, want queued relays held behind the running turn", got)
	}
}

func TestRelaySessionChatTurnDuplicateReportsNoOutcome(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.svc.RelaySessionChatTurn(ctx, testSession, "once", "relay-dup", ports.MessageDeliveryOptions{}); err != nil {
		t.Fatal(err)
	}
	again, err := h.svc.RelaySessionChatTurn(ctx, testSession, "once", "relay-dup", ports.MessageDeliveryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again != (ports.MessageDeliveryOutcome{}) {
		t.Fatalf("duplicate relay = %+v, want no new delivery", again)
	}
}
