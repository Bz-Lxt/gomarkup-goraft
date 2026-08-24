package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: raftctl <base-url> put|get|kill|revive|status <args>\n")
		os.Exit(2)
	}
	base := os.Args[1]
	cmd := os.Args[2]
	cli := &http.Client{Timeout: 3 * time.Second}
	switch cmd {
	case "status":
		dump(cli, http.MethodGet, base+"/api/v1/observe/state", nil)
	case "get":
		if len(os.Args) < 4 {
			os.Exit(2)
		}
		dump(cli, http.MethodGet, base+"/api/v1/kv/"+os.Args[3], nil)
	case "put":
		if len(os.Args) < 5 {
			os.Exit(2)
		}
		body, _ := json.Marshal(map[string]string{"value": os.Args[4], "client_id": "raftctl"})
		dump(cli, http.MethodPut, base+"/api/v1/kv/"+os.Args[3], body)
	case "kill":
		dump(cli, http.MethodPost, base+"/api/v1/chaos/kill", []byte("{}"))
	case "revive":
		dump(cli, http.MethodPost, base+"/api/v1/chaos/revive", []byte("{}"))
	default:
		fmt.Fprintf(os.Stderr, "unknown command %s\n", cmd)
		os.Exit(2)
	}
}

func dump(cli *http.Client, method, url string, body []byte) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	fmt.Printf("%s\n%s\n", resp.Status, string(b))
	if resp.StatusCode >= 400 {
		os.Exit(1)
	}
}
