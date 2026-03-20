# Example Reviewer Focus

- verify scoring weights are config-driven
- verify decay uses event timestamps rather than wall clock only
- verify deterministic tests for bullish/bearish/conflicting cases
- verify no direct coupling from signal-engine-service to collector-service
- verify `signals.generated` payload semantics remain unchanged
