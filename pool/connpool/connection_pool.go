package connpool

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// Pool 为连接提供了一个池功能，支持连接重用  全局连接池对象是所有协程共用的。它主要是实现对所有子连接池的统一管理
type Pool interface {
	Get(ctx context.Context, network string, address string) (net.Conn, error)
}

// Pool 的实现
type pool struct {
	opts *Options
	conns *sync.Map  //key 是 server 的监听地址，value 是子连接池
}

var poolMap = make(map[string]Pool)
var oneByte = make([]byte, 1)

func init() {
	registorPool("default", DefaultPool)
}

func registorPool(poolName string, pool Pool) {
	poolMap[poolName] = pool
}

// GetPool get a Pool by a pool name
func GetPool(poolName string) Pool {
	if v, ok := poolMap[poolName]; ok {
		return v
	}
	return DefaultPool
}

// TODO expose the ConnPool options
var DefaultPool = NewConnPool()

func NewConnPool(opt ...Option) *pool {
	// default options
	opts := &Options {
		maxCap: 1000,
		idleTimeout: 1 * time.Minute,
		dialTimeout: 200 * time.Millisecond,
	}
	m := &sync.Map{}

	p := &pool {
		conns : m,
		opts : opts,
	}
	for _, o := range opt {
		o(p.opts)
	}

	return p
}

// 获取连接 net.Conn
func (p *pool) Get(ctx context.Context, network string, address string) (net.Conn, error) {

	//从 map 中取出 key 为 address 的子连接池
	if value, ok := p.conns.Load(address); ok {

		// 断言 value 是 *channelPool 类型
		if cp, ok := value.(*channelPool); ok {
			conn, err := cp.Get(ctx) // 从连接池中获取连接
			return conn, err
		}
	}

	//假如不存在，那么说明是第一次调用，创建 后端 server 地址为 address 的子连接池
	cp, err := p.NewChannelPool(ctx, network, address)
	if err != nil {
		return nil, err
	}

	// 存储
	p.conns.Store(address, cp)

	return cp.Get(ctx)
}

// 子连接池
type channelPool struct {
	net.Conn
	initialCap int  // 初始容量
	maxCap int      // 最大容量
	maxIdle int     // 最大空闲连接数
	idleTimeout time.Duration  // 空闲超时时间
	dialTimeout time.Duration  // 发送超时时间
	Dial func(context.Context) (net.Conn, error)
	conns chan *PoolConn
	mu sync.RWMutex
}

// 初始化子连接池
func (p *pool) NewChannelPool(ctx context.Context, network string, address string) (*channelPool, error){
	c := &channelPool {
		initialCap: p.opts.initialCap, //指定初始化连接池的容量
		maxCap: p.opts.maxCap,
		Dial : func(ctx context.Context) (net.Conn, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			timeout := p.opts.dialTimeout
			if t , ok := ctx.Deadline(); ok {
				timeout = t.Sub(time.Now())
			}

			return net.DialTimeout(network, address, timeout)
		},
		conns : make(chan *PoolConn, p.opts.maxCap), //指定连接池最大容量
		idleTimeout: p.opts.idleTimeout,
		dialTimeout: p.opts.dialTimeout,
	}

	if p.opts.initialCap == 0 {
		// default initialCap is 1
		p.opts.initialCap = 1
	}

	// 连接池填充(根据 initialCap )数量填充
	for i := 0; i < p.opts.initialCap; i++ {
		conn , err := c.Dial(ctx);
		if err != nil {
			return nil, err
		}
		c.Put(c.wrapConn(conn))
	}

	//注册 连接检查 (每3秒进行一次)
	c.RegisterChecker(3 * time.Second, c.Checker)
	return c, nil
}

func (c *channelPool) Get(ctx context.Context) (net.Conn, error) {
	if c.conns == nil {
		return nil, ErrConnClosed
	}
	select {
		case pc := <-c.conns :
			if pc == nil {
				return nil, ErrConnClosed
			}

			if pc.unusable {
				return nil, ErrConnClosed
			}

			return pc, nil
		default:
			conn, err := c.Dial(ctx)
			if err != nil {
				return nil, err
			}
			return c.wrapConn(conn), nil
	}
}

func (c *channelPool) Close() {
	c.mu.Lock()
	conns := c.conns
	c.conns = nil
	c.Dial = nil
	c.mu.Unlock()

	if conns == nil {
		return
	}
	close(conns)
	for conn := range conns {
		conn.MarkUnusable()
		conn.Close()
	}
}

func (c *channelPool) Put(conn *PoolConn) error {
	if conn == nil {
		return errors.New("connection closed")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conns == nil {
		conn.MarkUnusable()
		conn.Close()
	}

	select {
	case c.conns <- conn :
		return nil
	default:
		// 连接池满
		return conn.Close()
	}
}

func (c *channelPool) RegisterChecker(internal time.Duration, checker func(conn *PoolConn) bool) {

	if internal <= 0 || checker == nil {
		return
	}

	go func() {

		for {

			time.Sleep(internal)

			length := len(c.conns)

			for i:=0; i < length; i++ {

				select {
				case pc := <- c.conns :

					if !checker(pc) {
						pc.MarkUnusable()
						pc.Close()
						break
					} else {
						c.Put(pc)
					}
				default:
					break
				}

			}
		}

	}()
}

// 健康检查函数
func (c *channelPool) Checker (pc *PoolConn) bool {

	// check timeout
	if pc.t.Add(c.idleTimeout).Before(time.Now()) {
		return false
	}

	// check conn is alive or not
	if !isConnAlive(pc.Conn) {
		return false
	}

	return true
}

// 检查连接是否存活
func isConnAlive(conn net.Conn) bool {
	conn.SetReadDeadline(time.Now().Add(time.Millisecond))

	//读取1个byte 如果 返回的数量 0 或者 是EOF 就返回失败
	if n, err := conn.Read(oneByte); n > 0 || err == io.EOF {
		return false
	}

	conn.SetReadDeadline(time.Time{})
	return true
}




