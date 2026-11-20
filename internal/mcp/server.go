package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/samam0234/my-go-gin-study/internal/store"
)

const ProtocolVersion = "2025-06-18"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// Handle 는 MCP JSON-RPC 한 건을 처리한다.
// 알림(notification)이면 response 가 nil 이다.
func Handle(ctx context.Context, st *store.Store, body []byte) ([]byte, error) {
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return rpcError(nil, -32700, "JSON을 읽을 수 없습니다"), nil
	}
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return rpcError(req.ID, -32600, "jsonrpc 버전은 2.0 이어야 합니다"), nil
	}
	switch req.Method {
	case "notifications/initialized":
		return nil, nil
	case "initialize":
		return rpcResult(req.ID, map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities": map[string]any{
				"tools":     map[string]any{"listChanged": false},
				"resources": map[string]any{},
			},
			"serverInfo": map[string]any{"name": "gin-board", "version": "0.1.0"},
		})
	case "ping":
		return rpcResult(req.ID, map[string]any{})
	case "tools/list":
		return rpcResult(req.ID, map[string]any{"tools": toolDefs()})
	case "tools/call":
		return callTool(ctx, st, req)
	case "resources/list":
		return rpcResult(req.ID, map[string]any{"resources": []map[string]any{statsResource()}})
	case "resources/read":
		return readResource(ctx, st, req)
	default:
		return rpcError(req.ID, -32601, "없는 메서드입니다"), nil
	}
}

func ServeStdio(ctx context.Context, st *store.Store) error {
	in := bufio.NewReader(os.Stdin)
	out := json.NewEncoder(os.Stdout)
	for {
		line, err := in.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		resp, err := Handle(ctx, st, line)
		if err != nil {
			return err
		}
		if resp == nil {
			continue
		}
		var payload any
		if err := json.Unmarshal(resp, &payload); err != nil {
			return err
		}
		if err := out.Encode(payload); err != nil {
			return err
		}
	}
}

func toolDefs() []map[string]any {
	return []map[string]any{
		{
			"name":        "search_posts",
			"description": "게시판 글을 제목·본문으로 검색한다. 최대 20개.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"q":     map[string]any{"type": "string", "description": "검색어"},
					"limit": map[string]any{"type": "integer", "description": "개수. 기본 10, 최대 20"},
				},
				"required":             []string{"q"},
				"additionalProperties": false,
			},
		},
		{
			"name":        "get_post",
			"description": "글 하나를 id로 가져온다. 읽기 전용.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"postId": map[string]any{"type": "integer"},
				},
				"required":             []string{"postId"},
				"additionalProperties": false,
			},
		},
	}
}

func statsResource() map[string]any {
	return map[string]any{
		"uri":         "board://stats",
		"name":        "stats",
		"description": "게시글 수",
		"mimeType":    "text/plain",
	}
}

func callTool(ctx context.Context, st *store.Store, req Request) ([]byte, error) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return rpcError(req.ID, -32602, "tools/call 파라미터가 잘못되었습니다"), nil
	}
	switch params.Name {
	case "search_posts":
		var args struct {
			Q     string `json:"q"`
			Limit int    `json:"limit"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if args.Limit < 1 {
			args.Limit = 10
		}
		if args.Limit > 20 {
			args.Limit = 20
		}
		rows, _, err := st.ListPosts(ctx, store.PostFilter{Q: args.Q, Page: 1, Limit: args.Limit, Sort: "-createdAt"})
		if err != nil {
			return toolText(req.ID, err.Error(), true)
		}
		type item struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
		}
		out := make([]item, 0, len(rows))
		for _, row := range rows {
			out = append(out, item{ID: row.ID, Title: row.Title})
		}
		raw, _ := json.Marshal(out)
		return toolText(req.ID, string(raw), false)
	case "get_post":
		var args struct {
			PostID int64 `json:"postId"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil || args.PostID < 1 {
			return rpcError(req.ID, -32602, "postId가 필요합니다"), nil
		}
		post, err := st.GetPost(ctx, args.PostID)
		if errors.Is(err, store.ErrNotFound) {
			return toolText(req.ID, "글을 찾을 수 없습니다", true)
		}
		if err != nil {
			return toolText(req.ID, err.Error(), true)
		}
		raw, _ := json.Marshal(map[string]any{
			"id": post.ID, "title": post.Title, "body": post.Body, "author": post.AuthorNickname,
		})
		return toolText(req.ID, string(raw), false)
	default:
		return rpcError(req.ID, -32602, "없는 도구입니다"), nil
	}
}

func readResource(ctx context.Context, st *store.Store, req Request) ([]byte, error) {
	var params struct {
		URI string `json:"uri"`
	}
	_ = json.Unmarshal(req.Params, &params)
	if params.URI != "board://stats" {
		return rpcError(req.ID, -32002, "없는 리소스입니다"), nil
	}
	n, err := st.CountPosts(ctx)
	if err != nil {
		return rpcError(req.ID, -32603, "통계를 읽지 못했습니다"), nil
	}
	return rpcResult(req.ID, map[string]any{
		"contents": []map[string]any{{
			"uri":      "board://stats",
			"mimeType": "text/plain",
			"text":     "posts=" + itoa(n),
		}},
	})
}

func toolText(id json.RawMessage, text string, isError bool) ([]byte, error) {
	return rpcResult(id, map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isError,
	})
}

func rpcResult(id json.RawMessage, result any) ([]byte, error) {
	if len(id) == 0 {
		id = []byte("null")
	}
	return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(id), "result": result})
}

func rpcError(id json.RawMessage, code int, message string) []byte {
	if len(id) == 0 {
		id = []byte("null")
	}
	raw, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      json.RawMessage(id),
		"error":   map[string]any{"code": code, "message": message},
	})
	return raw
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
