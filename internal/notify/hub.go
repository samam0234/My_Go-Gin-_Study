package notify

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *client) write(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

// Hub 는 로그인한 브라우저에 같은 알림을 보낸다.
type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
}

func New() *Hub {
	return &Hub{clients: map[*client]struct{}{}}
}

func (h *Hub) Add(conn *websocket.Conn) *client {
	c := &client{conn: conn}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) Remove(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	_ = c.conn.Close()
}

func (h *Hub) Broadcast(msg []byte) {
	h.mu.Lock()
	list := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		list = append(list, c)
	}
	h.mu.Unlock()
	for _, c := range list {
		if err := c.write(msg); err != nil {
			h.Remove(c)
		}
	}
}
