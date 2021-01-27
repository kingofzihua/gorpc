package connpool

import (
	"errors"
	"net"
	"sync"
	"time"
)

var (
	ErrConnClosed = errors.New("connection closed ...")
)
// 具体的连接类 PoolConn
type PoolConn struct {
	net.Conn
	c *channelPool
	unusable bool		// 如果 unusable 是 true 应该关闭连接
	mu sync.RWMutex
	t time.Time  // 连接空闲时间
	dialTimeout time.Duration // 连接超时持续时间
}

// overwrite conn Close for connection reuse
func (p *PoolConn) Close() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 如果连接不可用就关闭连接
	if p.unusable {
		if p.Conn != nil {
			return p.Conn.Close()
		}
	}

	// 重置连接截止时间
	p.Conn.SetDeadline(time.Time{})

	return p.c.Put(p)
}

// 标记连接为不可用
func (p *PoolConn) MarkUnusable() {
	p.mu.Lock()
	p.unusable = true
	p.mu.Unlock()
}

func (p *PoolConn) Read(b []byte) (int, error) {
	//如果连接是不可用状态就返回连接关闭
	if p.unusable {
		return 0, ErrConnClosed
	}
	n, err := p.Conn.Read(b)
	if err != nil {
		p.MarkUnusable()
		p.Conn.Close()
	}
	return n, err
}

func (p *PoolConn) Write(b []byte) (int, error) {
	if p.unusable {
		return 0, ErrConnClosed
	}
	n, err := p.Conn.Write(b)
	if err != nil {
		p.MarkUnusable()
		p.Conn.Close()
	}
	return n, err
}

func (c *channelPool) wrapConn(conn net.Conn) *PoolConn {
	p := &PoolConn {
		c : c,
		t : time.Now(),
		dialTimeout: c.dialTimeout,
	}
	p.Conn = conn
	return p
}

