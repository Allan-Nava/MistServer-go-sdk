package mist_go_test

import (
	"encoding/json"
	"fmt"
	"log"

	mist "github.com/Allan-Nava/MistServer-go-sdk/mist"
)

// ---------------------------------------------------------------- streams

func ExampleStreamsRequest() {
	client := newClient()

	// Replaces the whole list: streams not listed here are deleted.
	_, err := client.PostStreams(mist.StreamsRequest{Streams: map[string]mist.AddStream{
		"live": {Name: "live", Source: "push://"},
		"vod":  {Name: "vod", Source: "/media/intro.mp4"},
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("stream list replaced")
	// Output: stream list replaced
}

func ExampleDeleteStreamSourceRequest() {
	client := newClient()

	resp, err := client.PostDeleteStreamSource(mist.DeleteStreamSourceRequest{DeleteStreamSource: []string{"vod"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.DeleteStreamSource[0])
	// Output: -2: Stream and source file deleted
}

func ExampleNukeStreamRequest() {
	client := newClient()

	if err := client.PostNukeStream(mist.NukeStreamRequest{NukeStream: "live"}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("stream nuked")
	// Output: stream nuked
}

func ExampleNewService_noUnconfiguredStreams() {
	client := newClient()

	if err := client.PostNoUnconfiguredStreams(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("unconfigured streams stopped")
	// Output: unconfigured streams stopped
}

func ExampleTagStreamRequest() {
	client := newClient()

	// Stream tags can drive auto-pushes and triggers, e.g. only record some
	// streams of a wildcard group.
	err := client.PostTagStream(mist.TagStreamRequest{TagStream: map[string][]string{"live": {"record"}}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("tagged")
	// Output: tagged
}

func ExampleUntagStreamRequest() {
	client := newClient()

	err := client.PostUntagStream(mist.UntagStreamRequest{UntagStream: map[string][]string{"live": {"record"}}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("untagged")
	// Output: untagged
}

func ExampleStreamTagsRequest() {
	client := newClient()

	resp, err := client.PostStreamTags(mist.StreamTagsRequest{StreamTags: []string{"live"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.StreamTags["live"])
	// Output: [record]
}

// ---------------------------------------------------------------- statistics

func ExampleActiveStreamsRequest() {
	client := newClient()

	resp, err := client.PostActiveStreams(mist.ActiveStreamsRequest{Fields: []string{"viewers", "lastms"}})
	if err != nil {
		log.Fatal(err)
	}
	for name, fields := range resp.ActiveStreams {
		fmt.Println(name, "viewers:", fields["viewers"])
	}
	// Output: live viewers: 3
}

func ExampleStatsStreamsRequest() {
	client := newClient()

	resp, err := client.PostStatsStreams(mist.StatsStreamsRequest{Fields: []string{"clients"}})
	if err != nil {
		log.Fatal(err)
	}
	for _, name := range resp.StatsStreams.Names {
		fmt.Println(name, "clients:", resp.StatsStreams.Values[name][0])
	}
	// Output: live clients: 2
}

func ExampleClientsRequest() {
	client := newClient()

	// Data slightly in the past is the most complete.
	resp, err := client.PostClients(mist.ClientsRequest{Clients: mist.ClientsQuery{
		Streams: []string{"live"},
		Fields:  []string{"host", "protocol"},
		Time:    -30,
	}})
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range resp.Clients.Data {
		fmt.Println(row[0], row[1])
	}
	// Output: 203.0.113.7 HLS
}

func ExampleTotalsRequest() {
	client := newClient()

	resp, err := client.PostTotals(mist.TotalsRequest{Totals: mist.TotalsQuery{
		Fields: []string{"clients"},
		Start:  -60, // last minute
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(resp.Totals.Data), "points, fields", resp.Totals.Fields)
	// Output: 2 points, fields [clients]
}

func ExampleProcListRequest() {
	client := newClient()

	resp, err := client.PostProcList(mist.ProcListRequest{ProcList: "live"})
	if err != nil {
		log.Fatal(err)
	}
	for pid, p := range resp.ProcList {
		fmt.Printf("%s: %s %s -> %s\n", pid, p.Process, p.Source, p.Sink)
	}
	// Output: 4321: AV live -> live_720p
}

func ExampleCapabilitiesResponse() {
	client := newClient()

	resp, err := client.PostCapabilities()
	if err != nil {
		log.Fatal(err)
	}
	cp := resp.Capabilities
	fmt.Printf("cpu %.1f%%, %d/%d MiB used\n", float64(cp.CPUUse)/10, cp.Mem.Used, cp.Mem.Total)
	fmt.Println(len(cp.Connectors), "outputs,", len(cp.Inputs), "inputs")
	// Output:
	// cpu 12.5%, 3370/7898 MiB used
	// 2 outputs, 1 inputs
}

// ---------------------------------------------------------------- pushes

func ExamplePushStartRequest() {
	client := newClient()

	err := client.PostPushStart(mist.PushStartRequest{PushStart: mist.PushStart{
		Stream: "live",
		Target: "/recordings/$stream_$datetime.mkv",
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("push started")
	// Output: push started
}

func ExamplePushAutoListResponse() {
	client := newClient()

	resp, err := client.PostPushAutoList()
	if err != nil {
		log.Fatal(err)
	}
	for id, p := range resp.AutoPush {
		fmt.Println(id, p.Stream, "->", p.Target)
	}
	// Output: 71d6b51b live -> /rec/$stream.mkv
}

func ExamplePushSettingsRequest() {
	client := newClient()

	wait := 5 // retry a failed auto-push after 5 seconds
	resp, err := client.PostPushSettings(mist.PushSettingsRequest{PushSettings: mist.PushSettingsUpdate{Wait: &wait}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("wait:", resp.PushSettings.Wait, "maxspeed:", resp.PushSettings.MaxSpeed)
	// Output: wait: 5 maxspeed: 0
}

// ---------------------------------------------------------------- sessions

func ExampleStopSessionsRequest() {
	client := newClient()

	err := client.PostStopSessions(mist.StopSessionsRequest{StopSessions: map[string]string{
		"live": "",     // every session of "live"
		"":     "RTMP", // every RTMP session, on any stream
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("sessions stopped")
	// Output: sessions stopped
}

func ExampleStopSessIDRequest() {
	client := newClient()

	if err := client.PostStopSessID(mist.StopSessIDRequest{StopSessID: []string{"a1b2c3"}}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("session stopped")
	// Output: session stopped
}

func ExampleStopTagRequest() {
	client := newClient()

	if err := client.PostStopTag(mist.StopTagRequest{StopTag: []string{"banned"}}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("tagged sessions stopped")
	// Output: tagged sessions stopped
}

func ExampleTagSessIDRequest() {
	client := newClient()

	if err := client.PostTagSessID(mist.TagSessIDRequest{TagSessID: map[string]string{"a1b2c3": "banned"}}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("session tagged")
	// Output: session tagged
}

func ExampleInvalidateSessionsRequest() {
	client := newClient()

	// Makes USER_NEW run again for every viewer of "live".
	if err := client.PostInvalidateSessions(mist.InvalidateSessionsRequest{InvalidateSessions: []string{"live"}}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("sessions invalidated")
	// Output: sessions invalidated
}

// ---------------------------------------------------------------- configuration

func ExampleConfigRequest() {
	client := newClient()

	resp, err := client.PostConfig(mist.ConfigRequest{Config: map[string]any{
		"serverid":  "edge-milan-1",
		"accesslog": "LOG",
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("server:", resp.Config.Version)
	// Output: server: 3.4 Generic_64
}

func ExampleAddProtocolRequest() {
	client := newClient()

	_, err := client.PostAddProtocol(mist.AddProtocolRequest{AddProtocol: []mist.Protocol{
		{"connector": "RTMP", "port": 1935},
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("RTMP enabled")
	// Output: RTMP enabled
}

func ExampleDeleteProtocolRequest() {
	client := newClient()

	// Must match the configured output exactly.
	_, err := client.PostDeleteProtocol(mist.DeleteProtocolRequest{DeleteProtocol: []mist.Protocol{
		{"connector": "RTMP", "port": 1935},
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("RTMP removed")
	// Output: RTMP removed
}

func ExampleUpdateProtocolRequest() {
	client := newClient()

	_, err := client.PostUpdateProtocol(mist.UpdateProtocolRequest{
		Old: mist.Protocol{"connector": "RTMP", "port": 1935},
		New: mist.Protocol{"connector": "RTMP", "port": 1936},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("RTMP moved to 1936")
	// Output: RTMP moved to 1936
}

func ExampleConfigBackupResponse() {
	client := newClient()

	backup, err := client.PostConfigBackup()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(backup.ConfigBackup))

	// ...and put it back later.
	if err := client.PostConfigRestore(mist.ConfigRestoreRequest{ConfigRestore: backup.ConfigBackup}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("restored")
	// Output:
	// {"streams":{"live":{"source":"push://"}}}
	// restored
}

func ExampleConfigRestoreRequest() {
	client := newClient()

	cfg := json.RawMessage(`{"streams":{"live":{"source":"push://"}}}`)
	if err := client.PostConfigRestore(mist.ConfigRestoreRequest{ConfigRestore: cfg}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("restored")
	// Output: restored
}

func ExampleNewService_save() {
	client := newClient()

	// Persist configuration changes now instead of waiting for the controller.
	if err := client.PostSave(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("saved")
	// Output: saved
}

func ExampleUISettingsRequest() {
	client := newClient()

	// A nil map only reads the stored settings.
	resp, err := client.PostUISettings(mist.UISettingsRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(resp.UISettings))
	// Output: {"theme":"dark"}
}

func ExampleAPIEndpointResponse() {
	client := newClient()

	resp, err := client.PostAPIEndpoint()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.APIEndpoint)
	// Output: http://127.0.0.1:4242/
}

func ExampleBrowseRequest() {
	client := newClient()

	resp, err := client.PostBrowse(mist.BrowseRequest{Browse: "/media"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Browse.Dir(), resp.Browse.Files, resp.Browse.Subdirectories)
	// Output: /media [a.mp4 b.mkv] [archive]
}

func ExampleShutdownRequest() {
	client := newClient()

	resp, err := client.PostShutdown(mist.ShutdownRequest{Shutdown: "maintenance window"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Shutdown) // only a local connection may shut MistServer down
	// Output: Ignored - only local users may request shutdown
}

func ExampleNewService_logout() {
	client := newClient()

	if err := client.PostLogout(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("logged out")
	// Output: logged out
}

func ExampleNewService_clearStatLogs() {
	client := newClient()

	if err := client.PostClearStatLogs(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("log cleared")
	// Output: log cleared
}

func ExampleUpdateResponse() {
	client := newClient()

	resp, err := client.PostUpdate()
	if err != nil {
		log.Fatal(err)
	}
	if resp.Update.UpToDate == 0 {
		fmt.Println("update available:", resp.Update.Version)
	}
	// Output: update available: 3.5
}

func ExampleUpdateResponse_autoUpdate() {
	client := newClient()

	resp, err := client.PostAutoUpdate()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("updating to %s: %d%%\n", resp.Update.Version, resp.Update.Progress)
	// Output: updating to 3.5: 1%
}

// ---------------------------------------------------------------- stream keys and JWKs

func ExampleStreamKeysRequest() {
	client := newClient()

	// A nil map only reads; a non-nil map replaces every key.
	resp, err := client.PostStreamKeys(mist.StreamKeysRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.StreamKeys)
	// Output: map[UhFQ4DSY:live]
}

func ExampleStreamKeyAddRequest() {
	client := newClient()

	resp, err := client.PostStreamKeyAdd(mist.StreamKeyAddRequest{StreamKeyAdd: map[string]string{"UhFQ4DSY": "live"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("added:", resp.StreamKeyAdd.Added)
	// Output: added: [UhFQ4DSY]
}

func ExampleStreamKeyDelRequest() {
	client := newClient()

	resp, err := client.PostStreamKeyDel(mist.StreamKeyDelRequest{StreamKeyDel: []string{"UhFQ4DSY"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("deleted:", resp.StreamKeyDel.Deleted)
	// Output: deleted: [UhFQ4DSY]
}

func ExampleJWKSRequest() {
	client := newClient()

	// A nil slice only reads the configured keys.
	resp, err := client.PostJWKS(mist.JWKSRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(resp.JWKS), "key(s)")
	// Output: 1 key(s)
}

func ExampleAddJWKSRequest() {
	client := newClient()

	resp, err := client.PostAddJWKS(mist.AddJWKSRequest{AddJWKS: []mist.JWK{
		{
			Key:         "https://issuer.example.com/.well-known/jwks.json",
			Permissions: &mist.JWKPermissions{Output: true, Stream: "*"},
		},
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(resp.AddJWKS), "key(s) added")
	// Output: 1 key(s) added
}

func ExampleDeleteJWKSRequest() {
	client := newClient()

	// Match by key ID.
	resp, err := client.PostDeleteJWKS(mist.DeleteJWKSRequest{DeleteJWKS: []mist.JWK{
		{Key: map[string]any{"kid": "k1"}},
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(resp.DeleteJWKS))
	// Output: [{"kid":"k1"}]
}

// ---------------------------------------------------------------- variables and external writers

func ExampleVariablesResponse() {
	client := newClient()

	resp, err := client.PostVariableList()
	if err != nil {
		log.Fatal(err)
	}
	for name, v := range resp.VariableList {
		fmt.Printf("$%s = %s\n", name, v.Value)
	}
	// Output: $region = eu-south
}

func ExampleVariableAddRequest() {
	client := newClient()

	// Static: only a value. Dynamic: a Target command or URL plus Interval.
	resp, err := client.PostVariableAdd(mist.VariableAddRequest{VariableAdd: mist.Variable{Name: "region", Value: "eu-south"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.VariableList["region"].Value)
	// Output: eu-south
}

func ExampleVariableRemoveRequest() {
	client := newClient()

	resp, err := client.PostVariableRemove(mist.VariableRemoveRequest{VariableRemove: []string{"region"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(resp.VariableList), "variables left")
	// Output: 0 variables left
}

func ExampleExternalWritersResponse() {
	client := newClient()

	resp, err := client.PostExternalWriterList()
	if err != nil {
		log.Fatal(err)
	}
	for _, w := range resp.ExternalWriterList {
		fmt.Println(w.Name, w.CmdLine, w.Protocols)
	}
	// Output: s3 /usr/local/bin/s3-upload [s3]
}

func ExampleExternalWriterAddRequest() {
	client := newClient()

	// Pushes to s3://... are handed to the command.
	_, err := client.PostExternalWriterAdd(mist.ExternalWriterAddRequest{ExternalWriterAdd: mist.ExternalWriter{
		Name:      "s3",
		CmdLine:   "/usr/local/bin/s3-upload",
		Protocols: []string{"s3"},
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("writer added")
	// Output: writer added
}

func ExampleExternalWriterRemoveRequest() {
	client := newClient()

	resp, err := client.PostExternalWriterRemove(mist.ExternalWriterRemoveRequest{ExternalWriterRemove: "s3"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(resp.ExternalWriterList), "writers left")
	// Output: 0 writers left
}

// ---------------------------------------------------------------- anything else

func ExampleNewService_raw() {
	client := newClient()

	// Commands without a method, or several commands in one request.
	reply, err := client.PostRaw(map[string]any{
		"push_stop_graceful": []int{412},
		"api_endpoint":       true,
	})
	if err != nil {
		log.Fatal(err)
	}
	var endpoint string
	if err := json.Unmarshal(reply["api_endpoint"], &endpoint); err != nil {
		log.Fatal(err)
	}
	fmt.Println(endpoint)
	// Output: http://127.0.0.1:4242/
}
