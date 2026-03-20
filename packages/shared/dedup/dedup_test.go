package dedup_test

import (
	"testing"

	"github.com/trader-1/trader-1/packages/shared/dedup"
)

func TestKeyFormatting(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"raw event key", dedup.RawEventKey("telegram", "msg-123"), "dedup:raw:telegram:msg-123"},
		{"normalized event key", dedup.NormalizedEventKey("evt-abc"), "dedup:norm:evt-abc"},
		{"signal key", dedup.SignalKey("sig-xyz"), "dedup:signal:sig-xyz"},
		{"risk key", dedup.RiskKey("sig-xyz"), "dedup:risk:sig-xyz"},
		{"explain key", dedup.ExplainKey("sig-xyz"), "dedup:explain:sig-xyz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
