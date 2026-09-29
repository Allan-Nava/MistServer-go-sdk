package mist_go

import (
	"encoding/json"
	"testing"
)

func TestDecodeLenient(t *testing.T) {
	t.Run("syntax errors are still returned", func(t *testing.T) {
		var s Stream
		if err := json.Unmarshal([]byte(`{"name": "live",`), &s); err == nil {
			t.Fatal("err = nil, want a syntax error")
		}
	})
	t.Run("a non-object value leaves the zero value", func(t *testing.T) {
		// e.g. {"incomplete list": 1} inside an addstream reply's streams.
		var streams map[string]Stream
		if err := json.Unmarshal([]byte(`{"incomplete list": 1, "live": {"name": "live"}}`), &streams); err != nil {
			t.Fatalf("err = %v", err)
		}
		if streams["live"].Name != "live" || streams["incomplete list"].Name != "" {
			t.Errorf("streams = %+v", streams)
		}
	})
	t.Run("only the mismatched field is dropped", func(t *testing.T) {
		var s Stream
		if err := json.Unmarshal([]byte(`{"NAME": "live", "online": "yes", "debug": 3}`), &s); err != nil {
			t.Fatalf("err = %v", err)
		}
		if s.Name != "live" || s.Online != 0 || s.Debug != 3 {
			t.Errorf("stream = %+v, want name and debug kept, online dropped", s)
		}
	})
}
