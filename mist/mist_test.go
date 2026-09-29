package mist_go

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Allan-Nava/MistServer-go-sdk/lib"
	"github.com/go-resty/resty/v2"
)

// fakeMist mimics the MistServer controller API login: a request without a
// valid authorize block gets status CHALL and the current challenge.
type fakeMist struct {
	mu        sync.Mutex
	username  string
	password  string
	challenge string
	status    int    // HTTP status to reply with; 0 means 200
	reply     string // raw JSON for authorized requests; empty means a minimal OK reply

	requests   int              // all requests received
	challenges int              // requests answered with CHALL
	commands   []map[string]any // authorized request bodies
}

func (f *fakeMist) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requests++
	if f.status != 0 {
		http.Error(w, "boom", f.status)
		return
	}

	// Like MistServer (controller_api.cpp), only an exact "application/json"
	// Content-Type makes the body the command; otherwise it's read from the
	// "command" form value, and an unparsable command is treated as empty.
	var raw string
	if r.Header.Get("Content-Type") == "application/json" {
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
	} else {
		raw = r.FormValue("command")
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(raw), &body)

	if f.username == "" { // no accounts configured: NOACC, and no challenge
		_ = json.NewEncoder(w).Encode(map[string]any{
			"authorize": map[string]any{"status": "NOACC"},
		})
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
	if f.reply != "" {
		_, _ = io.WriteString(w, f.reply)
		return
	}
	reply := map[string]any{"authorize": map[string]any{"status": "OK"}}
	if _, ok := body["push_list"]; ok {
		reply["push_list"] = [][]any{{1, "live", "rtmp://example/app"}}
	}
	_ = json.NewEncoder(w).Encode(reply)
}

func newTestClient(t *testing.T, f *fakeMist, password string) IMistGoClient {
	t.Helper()
	return newTestClientWith(t, f, password, nil)
}

func newTestClientWith(t *testing.T, f *fakeMist, password string, rc *resty.Client) IMistGoClient {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return NewService(rc, nil,
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

func TestContentTypeIsExactlyApplicationJSON(t *testing.T) {
	// MistServer ignores the body unless Content-Type is exactly
	// "application/json", so a client default with a charset must not leak in.
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
	rc := resty.New().SetHeader("Content-Type", "application/json; charset=utf-8")
	c := newTestClientWith(t, f, "secret", rc)

	if _, err := c.Health(); err != nil {
		t.Fatalf("Health: %v", err)
	}
}

func TestNoAccountIsNotRetried(t *testing.T) {
	// NOACC carries no challenge, so a retry can't succeed; on a local
	// connection MistServer has already run the command, so it would run twice.
	f := &fakeMist{challenge: "c1"}
	c := newTestClient(t, f, "secret")

	_, err := c.Health()
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if f.requests != 2 {
		t.Errorf("requests = %d, want 2 (challenge + one attempt, no retry)", f.requests)
	}
}

// healthReply is shaped like a real non-minimal MistServer reply: streams,
// config and log are always sent (controller_api.cpp).
const healthReply = `{
  "LTS": 1,
  "authorize": {"status": "OK", "local": true},
  "config": {
    "accesslog": "LOG",
    "controller": {"interface": null, "port": null, "username": null},
    "debug": null,
    "defaultStream": null,
    "iid": "abcdef",
    "limits": null,
    "location": {"lat": 45.46, "lon": 9.19, "name": "Milan"},
    "prometheus": "metrics",
    "protocols": [{"connector": "HTTP", "online": 1}, {"connector": "RTMP", "online": "Enabled"}],
    "serverid": "",
    "sessionInputMode": 14, "sessionOutputMode": 14, "sessionStreamInfoMode": 1,
    "sessionUnspecifiedMode": 0, "sessionViewerMode": 14,
    "time": 1790000000,
    "tknMode": 15,
    "triggers": null,
    "trustedproxy": [],
    "version": "3.4 Generic_64"
  },
  "streams": {
    "live": {"name": "live", "source": "push://", "online": 1, "stop_sessions": false,
             "processes": [], "tags": ["ingest"], "DVR": 30000},
    "vod":  {"name": "vod", "source": "/media/a.mp4", "online": 2, "error": "Available"}
  },
  "log": [[1790000000, "CONF", "Controller started", ""]]
}`

func TestHealthDecodesRealisticReply(t *testing.T) {
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1", reply: healthReply}
	c := newTestClient(t, f, "secret")

	resp, err := c.Health()
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if got := resp.Streams["live"].Online; got != 1 {
		t.Errorf("streams.live.online = %d, want 1", got)
	}
	if got := resp.Config.Version; got != "3.4 Generic_64" {
		t.Errorf("config.version = %q", got)
	}
	if len(resp.Log) != 1 {
		t.Errorf("log entries = %d, want 1", len(resp.Log))
	}
}

func TestConcurrentCallsShareOneLogin(t *testing.T) {
	f := &fakeMist{username: "admin", password: "secret", challenge: "c1"}
	c := newTestClient(t, f, "secret")

	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Health()
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Health: %v", err)
		}
	}
	if f.challenges != 1 {
		t.Errorf("challenge requests = %d, want 1", f.challenges)
	}
}
