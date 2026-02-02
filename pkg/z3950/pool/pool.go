package pool

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

// Config 连接池配置
type Config struct {
	MaxIdle     int           // 每个 Target 最大空闲连接数
	IdleTimeout time.Duration // 空闲超时时间
}

var DefaultConfig = Config{
	MaxIdle:     5,
	IdleTimeout: 5 * time.Minute,
}

// ClientWrapper 包装 z3950.Client，增加元数据
type ClientWrapper struct {
	Client   *z3950.Client
	Host     string
	Port     int
	DBName   string
	LastUsed time.Time
}

// Pool 管理多目标的连接池
type Pool struct {
	mu      sync.Mutex
	pools   map[string][]*ClientWrapper // key: "host:port:db"
	config  Config
}

var globalPool *Pool
var once sync.Once

// GetGlobalPool 获取全局单例
func GetGlobalPool() *Pool {
	once.Do(func() {
		globalPool = NewPool(DefaultConfig)
		go globalPool.cleanupLoop()
	})
	return globalPool
}

func NewPool(cfg Config) *Pool {
	return &Pool{
		pools:  make(map[string][]*ClientWrapper),
		config: cfg,
	}
}

func (p *Pool) genKey(host string, port int, db string) string {
	return fmt.Sprintf("%s:%d:%s", host, port, db)
}

// Get 从池中获取连接，如果没有则新建
func (p *Pool) Get(host string, port int, db string) (*ClientWrapper, error) {
	key := p.genKey(host, port, db)
	
	p.mu.Lock()
	conns := p.pools[key]
	
	if len(conns) > 0 {
		wrapper := conns[len(conns)-1]
		p.pools[key] = conns[:len(conns)-1]
		p.mu.Unlock()
		
		if time.Since(wrapper.LastUsed) > p.config.IdleTimeout {
			slog.Info("pool: connection expired, closing", "host", host)
			wrapper.Client.Close()
			return p.Get(host, port, db)
		}
		
		slog.Info("pool: hit", "host", host)
		return wrapper, nil
	}
	p.mu.Unlock()

	slog.Info("pool: miss, creating new connection", "host", host)
	client := z3950.NewClient(host, port)
	if err := client.Connect(); err != nil {
		return nil, err
	}
	if err := client.Init(); err != nil {
		client.Close()
		return nil, err
	}
	
	return &ClientWrapper{
		Client: client,
		Host:   host,
		Port:   port,
		DBName: db,
		LastUsed: time.Now(),
	}, nil
}

// Put 归还连接
func (p *Pool) Put(cw *ClientWrapper) {
	if cw == nil || cw.Client == nil {
		return
	}
	
	cw.LastUsed = time.Now()
	key := p.genKey(cw.Host, cw.Port, cw.DBName)
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	conns := p.pools[key]
	if len(conns) >= p.config.MaxIdle {
		slog.Info("pool: full, closing connection", "host", cw.Host)
		cw.Client.Close()
		return
	}
	
	p.pools[key] = append(conns, cw)
}

func (p *Pool) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		p.mu.Lock()
		now := time.Now()
		for key, conns := range p.pools {
			var valid []*ClientWrapper
			for _, cw := range conns {
				if now.Sub(cw.LastUsed) <= p.config.IdleTimeout {
					valid = append(valid, cw)
				} else {
					cw.Client.Close()
				}
			}
			p.pools[key] = valid
		}
		p.mu.Unlock()
	}
}