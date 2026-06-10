package memory

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type writeOp struct {
	key   string
	value []byte
	ts    int64
}

type MemoryBus struct {
	db        *sql.DB
	lru       *LRUCache
	mu        sync.RWMutex
	writeCh   chan writeOp
	emergency chan writeOp
	batch     []writeOp
	done      chan struct{}
	wg        sync.WaitGroup
}

func NewMemoryBus(dbPath string, cacheSize int) (*MemoryBus, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	for _, q := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA mmap_size=268435456",
	} {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("%s: %w", q[:20], err)
		}
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS memory (key TEXT PRIMARY KEY, value BLOB, ts INTEGER)`); err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_memory_ts ON memory(ts)`); err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}
	mb := &MemoryBus{
		db: db, lru: NewLRU(cacheSize),
		writeCh: make(chan writeOp, 1024), emergency: make(chan writeOp, 256),
		done: make(chan struct{}),
	}
	mb.wg.Add(2)
	go mb.batchWriter()
	go mb.emergencyWriter()
	return mb, nil
}

func (m *MemoryBus) Write(key string, value []byte, ts int64) {
	op := writeOp{key: key, value: value, ts: ts}
	m.mu.Lock()
	m.lru.Set(key, value)
	m.mu.Unlock()
	select {
	case m.writeCh <- op:
	default:
		select {
		case m.emergency <- op:
		default:
			log.Printf("[MemoryBus] Emergency write dropped for key: %s", key)
		}
	}
}

func (m *MemoryBus) Read(key string) ([]byte, bool) {
	m.mu.RLock()
	if v, ok := m.lru.Get(key); ok {
		m.mu.RUnlock()
		return v, true
	}
	m.mu.RUnlock()
	var v []byte
	if err := m.db.QueryRow("SELECT value FROM memory WHERE key=?", key).Scan(&v); err != nil {
		return nil, false
	}
	m.mu.Lock()
	m.lru.Set(key, v)
	m.mu.Unlock()
	return v, true
}

func (m *MemoryBus) batchWriter() {
	defer m.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			m.mu.Lock()
			for {
				select {
				case op := <-m.writeCh:
					m.batch = append(m.batch, op)
				default:
					goto flush
				}
			}
		flush:
			if len(m.batch) == 0 {
				m.mu.Unlock()
				continue
			}
			b := append([]writeOp(nil), m.batch...)
			m.batch = nil
			m.mu.Unlock()
			sort.Slice(b, func(i, j int) bool { return b[i].ts < b[j].ts })
			tx, _ := m.db.Begin()
			for _, op := range b {
				tx.Exec("INSERT OR REPLACE INTO memory(key,value,ts) VALUES(?,?,?)", op.key, op.value, op.ts)
			}
			tx.Commit()
		}
	}
}

func (m *MemoryBus) emergencyWriter() {
	defer m.wg.Done()
	for {
		select {
		case <-m.done:
			return
		case op := <-m.emergency:
			m.db.Exec("INSERT OR REPLACE INTO memory(key,value,ts) VALUES(?,?,?)", op.key, op.value, op.ts)
		}
	}
}

func (m *MemoryBus) Close() {
	close(m.done)
	m.wg.Wait()
	m.db.Close()
}
