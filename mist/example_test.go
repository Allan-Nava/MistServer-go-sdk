package mist_go_test

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	mist "github.com/Allan-Nava/MistServer-go-sdk/mist"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

func ExampleNewService() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	client := mist.NewService(
		resty.New(),                // nil → resty.New()
		logger.Sugar(),             // nil → no-op logger
		mist.WithBaseURL(apiURL()), // e.g. "http://localhost:4242/api"
		mist.WithUsername("admin"),
		mist.WithPassword("secret"),
	)

	resp, err := client.Health()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("login:", resp.Authorize.Status)
	fmt.Println("server:", resp.Config.Version)
	// Output:
	// login: OK
	// server: 3.4 Generic_64
}

func ExampleNewService_health() {
	client := newClient()

	resp, err := client.Health()
	if err != nil {
		log.Fatal(err)
	}
	for name, s := range resp.Streams {
		fmt.Printf("%s online=%d source=%s\n", name, s.Online, s.Source)
	}
	// Output:
	// live online=1 source=push://
}

func ExamplePostStreamRequest() {
	client := newClient()

	// Creates the stream, or updates it if it exists. Zero-valued fields
	// (DVR, Debug, StopSessions) are not sent, so they don't reset settings.
	_, err := client.PostStream(mist.PostStreamRequest{
		AddStream: map[string]mist.AddStream{
			"live": {Name: "live", Source: "push://", DVR: 30000},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("stream saved")
	// Output: stream saved
}

func ExamplePostStreamRemoveRequest() {
	client := newClient()

	if _, err := client.PostStreamRemove(mist.PostStreamRemoveRequest{DeleteStream: "live"}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("stream deleted")
	// Output: stream deleted
}

func ExamplePostAutoPushRequest() {
	client := newClient()

	// Restream "live" to the target every time it comes online.
	_, err := client.PostAutoPush(mist.PostAutoPushRequest{
		PushAutoAdd: mist.PushAutoAdd{
			Stream: "live",
			Target: "rtmp://ingest.example.com/app/KEY",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("auto-push added")
	// Output: auto-push added
}

func ExamplePostAutoPushRemoveRequest() {
	client := newClient()

	// A stream name removes every auto-push rule for that stream.
	if _, err := client.PostAutoPushRemove(mist.PostAutoPushRemoveRequest{PushAutoRemove: "live"}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("auto-pushes removed")
	// Output: auto-pushes removed
}

func ExamplePostPushListRequest() {
	client := newClient()

	resp, err := client.PostPushList(mist.PostPushListRequest{PushList: true})
	if err != nil {
		log.Fatal(err)
	}
	// Each entry is [id, stream, original target, resolved target, ...].
	for _, p := range resp.PushList {
		fmt.Printf("push %v: %v -> %v\n", p[0], p[1], p[3])
	}
	// Output:
	// push 412: live -> rtmp://ingest.example.com/app/KEY
}

func ExamplePostPushStopRequest() {
	client := newClient()

	pushes, err := client.PostPushList(mist.PostPushListRequest{PushList: true})
	if err != nil {
		log.Fatal(err)
	}

	var ids []int
	for _, p := range pushes.PushList {
		ids = append(ids, int(p[0].(float64))) // JSON numbers decode as float64
	}
	if _, err := client.PostPushStop(mist.PostPushStopRequest{PushStop: ids}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("stopped", ids)
	// Output: stopped [412]
}

func ExampleNewService_badCredentials() {
	client := mist.NewService(nil, nil,
		mist.WithBaseURL(apiURL()),
		mist.WithUsername("admin"),
		mist.WithPassword("wrong"),
	)

	_, err := client.Health()
	if errors.Is(err, mist.ErrUnauthorized) {
		fmt.Println("bad credentials")
	}
	// Output: bad credentials
}

// --- scaffolding: a minimal stand-in for a MistServer controller ---

func newClient() mist.IMistGoClient {
	return mist.NewService(nil, nil,
		mist.WithBaseURL(apiURL()),
		mist.WithUsername("admin"),
		mist.WithPassword("secret"),
	)
}

// cannedReplies maps a request member to the reply members the fake sends
// back for it, shaped like the MistServer controller's replies.
var cannedReplies = map[string]map[string]any{
	"push_list":          {"push_list": [][]any{{412, "live", "rtmp://ingest.example.com/app/KEY", "rtmp://ingest.example.com/app/KEY"}}},
	"deletestreamsource": {"deletestreamsource": []string{"-2: Stream and source file deleted"}},
	"stream_tags":        {"stream_tags": map[string]any{"live": []string{"record"}}},
	"active_streams":     {"active_streams": map[string]any{"live": map[string]any{"viewers": 3, "lastms": 120000}}},
	"stats_streams":      {"stats_streams": map[string]any{"live": []int{2}}},
	"clients": {"clients": map[string]any{"time": 1790000000, "fields": []string{"host", "protocol"},
		"data": [][]any{{"203.0.113.7", "HLS"}}}},
	"totals": {"totals": map[string]any{"start": 1790000000, "end": 1790000060, "fields": []string{"clients"},
		"interval": [][]int{{12, 5}}, "data": [][]int{{4}, {5}}}},
	"proc_list": {"proc_list": map[string]any{"4321": map[string]any{"source": "live", "sink": "live_720p", "process": "AV"}}},
	"capabilities": {"capabilities": map[string]any{"connectors": map[string]any{"HLS": map[string]any{}, "RTMP": map[string]any{}},
		"inputs": map[string]any{"Buffer": map[string]any{}}, "cpu_use": 125, "threads": 8,
		"mem": map[string]any{"total": 7898, "used": 3370}}},
	"push_auto_list": {"auto_push": map[string]any{"71d6b51b": map[string]any{"stream": "live", "target": "/rec/$stream.mkv"}}},
	"push_settings":  {"push_settings": map[string]any{"wait": 5, "maxspeed": 0}},
	"config_backup":  {"config_backup": map[string]any{"streams": map[string]any{"live": map[string]any{"source": "push://"}}}},
	"ui_settings":    {"ui_settings": map[string]any{"theme": "dark"}},
	"api_endpoint":   {"api_endpoint": "http://127.0.0.1:4242/"},
	"browse": {"browse": map[string]any{"path": []string{"/media"}, "files": []string{"a.mp4", "b.mkv"},
		"subdirectories": []string{"archive"}}},
	"shutdown":               {"shutdown": "Ignored - only local users may request shutdown"},
	"update":                 {"update": map[string]any{"version": "3.5", "uptodate": 0}},
	"autoupdate":             {"update": map[string]any{"version": "3.5", "uptodate": 0, "progress": 1}},
	"streamkeys":             {"streamkeys": map[string]string{"UhFQ4DSY": "live"}},
	"streamkey_add":          {"streamkey_add": map[string]any{"added": []string{"UhFQ4DSY"}}},
	"streamkey_del":          {"streamkey_del": map[string]any{"deleted": []string{"UhFQ4DSY"}}},
	"jwks":                   {"jwks": []any{[]any{map[string]any{"kid": "k1"}, map[string]any{"input": true, "stream": "*"}}}},
	"addjwks":                {"addjwks": []any{"https://issuer.example.com/.well-known/jwks.json"}},
	"deletejwks":             {"deletejwks": []any{map[string]any{"kid": "k1"}}},
	"variable_list":          {"variable_list": map[string]any{"region": map[string]any{"value": "eu-south"}}},
	"variable_add":           {"variable_list": map[string]any{"region": map[string]any{"value": "eu-south"}}},
	"variable_remove":        {"variable_list": map[string]any{}},
	"external_writer_list":   {"external_writer_list": []any{[]any{"s3", "/usr/local/bin/s3-upload", []string{"s3"}}}},
	"external_writer_add":    {"external_writer_list": []any{[]any{"s3", "/usr/local/bin/s3-upload", []string{"s3"}}}},
	"external_writer_remove": {"external_writer_list": []any{}},
}

// apiURL starts a fake MistServer and returns its API URL. It speaks just
// enough of the protocol for the examples: the challenge login and canned
// replies for each command.
func apiURL() string {
	const challenge = "0123456789abcdef"
	md5hex := func(s string) string { h := md5.Sum([]byte(s)); return hex.EncodeToString(h[:]) }
	want := md5hex(md5hex("secret") + challenge)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &req)

		auth, _ := req["authorize"].(map[string]any)
		if auth == nil || auth["username"] != "admin" || auth["password"] != want {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"authorize": map[string]any{"status": "CHALL", "challenge": challenge},
			})
			return
		}

		reply := map[string]any{
			"authorize": map[string]any{"status": "OK"},
			"config":    map[string]any{"version": "3.4 Generic_64"},
			"streams": map[string]any{
				"live": map[string]any{"name": "live", "source": "push://", "online": 1},
			},
		}
		for member := range req {
			for k, v := range cannedReplies[member] {
				reply[k] = v
			}
		}
		_ = json.NewEncoder(w).Encode(reply)
	}))
	return srv.URL + "/api"
}
