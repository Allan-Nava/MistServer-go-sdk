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

func ExampleIMistGoClient_Health() {
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

func ExampleIMistGoClient_PostStream() {
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

func ExampleIMistGoClient_PostStreamRemove() {
	client := newClient()

	if _, err := client.PostStreamRemove(mist.PostStreamRemoveRequest{DeleteStream: "live"}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("stream deleted")
	// Output: stream deleted
}

func ExampleIMistGoClient_PostAutoPush() {
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

func ExampleIMistGoClient_PostAutoPushRemove() {
	client := newClient()

	// A stream name removes every auto-push rule for that stream.
	if _, err := client.PostAutoPushRemove(mist.PostAutoPushRemoveRequest{PushAutoRemove: "live"}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("auto-pushes removed")
	// Output: auto-pushes removed
}

func ExampleIMistGoClient_PostPushList() {
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

func ExampleIMistGoClient_PostPushStop() {
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

func ExampleErrUnauthorized() {
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
		if _, ok := req["push_list"]; ok {
			reply["push_list"] = [][]any{{412, "live", "rtmp://ingest.example.com/app/KEY", "rtmp://ingest.example.com/app/KEY"}}
		}
		_ = json.NewEncoder(w).Encode(reply)
	}))
	return srv.URL + "/api"
}
