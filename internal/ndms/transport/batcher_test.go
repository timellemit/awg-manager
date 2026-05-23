package transport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestBatcher constructs a Batcher backed by a custom HTTP handler
// (mock NDMS). Returns the batcher and a cleanup func.
func newTestBatcher(t *testing.T, handler http.HandlerFunc, window time.Duration) (*Batcher, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	addr := strings.TrimPrefix(srv.URL, "http://")
	cli := NewWithURL("http://"+addr, NewSemaphore(30))
	b := newBatcher(cli, window, 64, 256)
	b.Start()
	cleanup := func() {
		b.Close()
		srv.Close()
	}
	return b, cleanup
}

// echoBatchHandler returns array of `[{"echo": <pathN>}]` for each item
// in the POST batch — позволяет тестам verify ordering и distribution.
func echoBatchHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "want POST", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var batch []json.RawMessage
		if err := json.Unmarshal(body, &batch); err != nil {
			http.Error(w, "bad batch", http.StatusBadRequest)
			return
		}
		responses := make([]json.RawMessage, len(batch))
		for i, item := range batch {
			responses[i] = json.RawMessage(`{"echo":` + string(item) + `}`)
		}
		out, _ := json.Marshal(responses)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	}
}

func TestBatcher_SingleSubmit_FlushesAfterWindow(t *testing.T) {
	b, cleanup := newTestBatcher(t, echoBatchHandler(t), 10*time.Millisecond)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	body, err := b.Submit(ctx, "/show/interface/")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !strings.Contains(string(body), `"echo"`) {
		t.Errorf("body = %s, want contains echo wrapper", string(body))
	}
}
