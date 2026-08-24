package config

import "testing"

func TestParsePeersAndValidate(t *testing.T) {
	p, err := ParsePeers("n1=a:1,n2=b:2")
	if err != nil || p["n1"] != "a:1" {
		t.Fatal(p, err)
	}
	if _, err := ParsePeers("bad"); err == nil {
		t.Fatal("expected error")
	}
	c := &Config{
		ID: "n1", HTTPAddr: ":1", AdvertiseAddr: "a", DataDir: "/tmp",
		Peers: map[string]string{"n1": "a"}, Mode: ModeDemo,
		TickInterval: 1, HeartbeatTicks: 2, ElectionTicksMin: 5, ElectionTicksMax: 8,
		WALSegmentBytes: 1 << 20, SnapshotChunkSize: 2048,
	}
	c.applyModeDefaults()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(c.PeerIDs()) != 1 {
		t.Fatal(c.PeerIDs())
	}
}
