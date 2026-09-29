package mist_go

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Allan-Nava/MistServer-go-sdk/lib"
)

// fakeMist mimics the MistServer controller API login: a request without a
// valid authorize block gets status CHALL and the current challenge.
type fakeMist struct {
	mu        sync.Mutex
	username  string
	password  string
	challenge string
	status    int // HTTP status to reply with; 0 means 200

	challenges int              // requests answered with CHALL
	commands   []map[string]any // authorized request bodies
}

func (f *fakeMist) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.status != 0 {
		http.Error(w, "boom", f.status)
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	auth, _ := body["authorize"].(map[string]any)
	want := lib.GenerateMD5(lib.GenerateMD5(f.password) + f.challenge)
	if auth == nil || auth["username"] != f.username || auth["password"] != want {
		f.challenges++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"authorize": map[string]any{"status": "CHALL", "challenge": f.challenge},
		})
		return
	}

	f.commands = append(f.commands, body)
	reply := map[string]any{"authorize": map[string]any{"status": "OK"}}
	if _, ok := body["push_list"]; ok {
		reply["push_list"] = [][]any{{1, "live", "rtmp://example/app"}}
	}
	_ = json.NewEncoder(w).Encode(reply)
}

func newTestClient(t *testing.T, f *fakeMist, password string) IMistGoClient {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return NewService(nil, nil,
		WithBaseURL(srv.URL),
		WithUsername("admin"),
		WithPassword(password),
	)
}

func TestAuthorizedCallIsCached(t *testing.T) {
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
	c := newTestClient(t, f, "secret")

	if _, err := c.Health(); err != nil {
		t.Fatalf("Health: %v", err)
	}
	resp, err := c.PostPushList(PostPushListRequest{PushList: true})
	if err != nil {
		t.Fatalf("PostPushList: %v", err)
	}

	if len(resp.PushList) != 1 {
		t.Errorf("push_list = %v, want one entry", resp.PushList)
	}
	if f.challenges != 1 {
		t.Errorf("challenge requests = %d, want 1 (login should be cached)", f.challenges)
	}
	if len(f.commands) != 2 {
		t.Errorf("authorized commands = %d, want 2", len(f.commands))
	}
}

func TestWrongPasswordReturnsErrUnauthorized(t *testing.T) {
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
	c := newTestClient(t, f, "wrong")

	_, err := c.Health()
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if len(f.commands) != 0 {
		t.Errorf("authorized commands = %d, want 0", len(f.commands))
	}
}

func TestExpiredChallengeIsRetried(t *testing.T) {
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
	c := newTestClient(t, f, "secret")

	if _, err := c.Health(); err != nil {
		t.Fatalf("first Health: %v", err)
	}

	f.mu.Lock()
	f.challenge = "c2" // server rotates the challenge; the cached hash is now stale
	f.mu.Unlock()

	if _, err := c.Health(); err != nil {
		t.Fatalf("Health after rotation: %v", err)
	}
	if len(f.commands) != 2 {
		t.Errorf("authorized commands = %d, want 2", len(f.commands))
	}
}

func TestHTTPErrorIsReturned(t *testing.T) {
	f := &fakeMist{status: http.StatusInternalServerError}
	c := newTestClient(t, f, "secret")

	if _, err := c.Health(); err == nil {
		t.Fatal("err = nil, want an error for HTTP 500")
	}
}

func TestAddStreamOmitsZeroValues(t *testing.T) {
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
	c := newTestClient(t, f, "secret")

	_, err := c.PostStream(PostStreamRequest{
		AddStream: map[string]AddStream{"live": {Name: "live", Source: "push://"}},
	})
	if err != nil {
		t.Fatalf("PostStream: %v", err)
	}

	stream := f.commands[0]["addstream"].(map[string]any)["live"].(map[string]any)
	for _, key := range []string{"DVR", "debug", "stop_sessions"} {
		if _, ok := stream[key]; ok {
			t.Errorf("addstream.live has %q, want it omitted when zero", key)
		}
	}
}
