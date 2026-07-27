// Command dudka-mcp is a stdio MCP server for home agents on LAN (P113–P114).
// Tools talk to a local dudkad loopback HTTP API (no WAN).
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"dudka/internal/agent"
	"dudka/internal/discovery"
)

func main() {
	engine := flag.String("engine", "http://127.0.0.1:17880", "loopback dudkad base URL")
	flag.Parse()
	base := strings.TrimRight(strings.TrimSpace(*engine), "/")
	if !strings.Contains(base, "://") {
		base = "http://" + base
	}
	if err := assertLoopbackOrLAN(base); err != nil {
		fmt.Fprintf(os.Stderr, "dudka-mcp: %v\n", err)
		os.Exit(1)
	}
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	for {
		line, err := in.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "dudka-mcp: read: %v\n", err)
			return
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var req map[string]any
		if err := json.Unmarshal(line, &req); err != nil {
			writeJSON(out, map[string]any{"jsonrpc": "2.0", "error": map[string]any{"code": -32700, "message": "parse error"}, "id": nil})
			continue
		}
		id := req["id"]
		method, _ := req["method"].(string)
		switch method {
		case "initialize":
			writeJSON(out, map[string]any{
				"jsonrpc": "2.0", "id": id,
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "dudka-mcp", "version": "0.1.0"},
				},
			})
		case "notifications/initialized", "initialized":
			// no response
		case "tools/list":
			writeJSON(out, map[string]any{
				"jsonrpc": "2.0", "id": id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "dudka_send",
							"description": "Send text into the apartment chat feed (agent → chat)",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"text":    map[string]any{"type": "string"},
									"channel": map[string]any{"type": "string"},
								},
								"required": []string{"text"},
							},
						},
						{
							"name":        "dudka_inbox",
							"description": "Poll recent chat messages (chat → agent)",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"channel": map[string]any{"type": "string"},
									"limit":   map[string]any{"type": "integer"},
								},
							},
						},
						{
							"name":        "dudka_set_agent_nick",
							"description": "Validate/set agent triple-prefix nick via POST /nick",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"agent": map[string]any{"type": "string"},
									"model": map[string]any{"type": "string"},
									"host":  map[string]any{"type": "string"},
								},
								"required": []string{"agent", "model", "host"},
							},
						},
					},
				},
			})
		case "tools/call":
			params, _ := req["params"].(map[string]any)
			name, _ := params["name"].(string)
			args, _ := params["arguments"].(map[string]any)
			if args == nil {
				args = map[string]any{}
			}
			text, err := callTool(base, name, args)
			if err != nil {
				writeJSON(out, map[string]any{
					"jsonrpc": "2.0", "id": id,
					"result": map[string]any{
						"content": []map[string]any{{"type": "text", "text": err.Error()}},
						"isError": true,
					},
				})
				continue
			}
			writeJSON(out, map[string]any{
				"jsonrpc": "2.0", "id": id,
				"result": map[string]any{
					"content": []map[string]any{{"type": "text", "text": text}},
				},
			})
		default:
			if id != nil {
				writeJSON(out, map[string]any{
					"jsonrpc": "2.0", "id": id,
					"error": map[string]any{"code": -32601, "message": "method not found"},
				})
			}
		}
	}
}

func writeJSON(w *bufio.Writer, v any) {
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
	_ = w.Flush()
}

func callTool(base, name string, args map[string]any) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	switch name {
	case "dudka_send":
		text, _ := args["text"].(string)
		ch, _ := args["channel"].(string)
		body, _ := json.Marshal(map[string]any{"text": text, "channel": ch})
		resp, err := client.Post(base+"/send", "application/json", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return "", fmt.Errorf("send %d: %s", resp.StatusCode, string(raw))
		}
		return string(raw), nil
	case "dudka_inbox":
		url := base + "/messages"
		if ch, _ := args["channel"].(string); ch != "" {
			url += "?channel=" + ch
		}
		resp, err := client.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return "", fmt.Errorf("messages %d: %s", resp.StatusCode, string(raw))
		}
		return string(raw), nil
	case "dudka_set_agent_nick":
		a, _ := args["agent"].(string)
		m, _ := args["model"].(string)
		h, _ := args["host"].(string)
		nick, err := agent.FormatAgentNick(a, m, h)
		if err != nil {
			return "", err
		}
		body, _ := json.Marshal(map[string]string{"name": nick})
		resp, err := client.Post(base+"/nick", "application/json", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return "", fmt.Errorf("nick %d: %s", resp.StatusCode, string(raw))
		}
		return nick, nil
	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}

func assertLoopbackOrLAN(base string) error {
	u := strings.TrimPrefix(base, "http://")
	u = strings.TrimPrefix(u, "https://")
	host, _, err := net.SplitHostPort(u)
	if err != nil {
		host = u
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() {
			return nil
		}
		return discovery.CheckDialHost(ip.String())
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	return discovery.CheckDialHost(host)
}
