package scheduler

import (
	"container/heap"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
)

type RoutingPolicy struct {
	TypePriority   map[bus.PacketType]int
	TypePrecedence map[bus.PacketType]int
	Fanout         bool
	DropIfBusy     bool
}

var DefaultRoutingPolicy = RoutingPolicy{
	TypePriority: map[bus.PacketType]int{
		bus.Perception: 90, bus.Thought: 80, bus.Meta: 70, bus.Memory: 60,
		bus.Action: 50, bus.Output: 95, bus.ToolRequest: 40, bus.ToolResult: 85,
	},
	TypePrecedence: map[bus.PacketType]int{
		bus.Perception: 1, bus.Memory: 2, bus.Thought: 3, bus.Meta: 4,
		bus.Action: 5, bus.Output: 6, bus.ToolRequest: 5, bus.ToolResult: 2,
	},
	Fanout: false, DropIfBusy: true,
}

type SchedulerMode int

const (
	ModeNormal        SchedulerMode = iota
	ModeDeterministic
)

type PacketHeap struct {
	packets   []bus.CognitivePacket
	scheduler *Scheduler
}

func (h *PacketHeap) Len() int { return len(h.packets) }

func (h *PacketHeap) Less(i, j int) bool {
	now := h.scheduler.now
	ea := effective(h.packets[i], now)
	eb := effective(h.packets[j], now)
	if ea != eb {
		return ea > eb
	}
	return h.packets[i].EnqueuedAt < h.packets[j].EnqueuedAt
}

func (h *PacketHeap) Swap(i, j int) { h.packets[i], h.packets[j] = h.packets[j], h.packets[i] }
func (h *PacketHeap) Push(x any)     { h.packets = append(h.packets, x.(bus.CognitivePacket)) }
func (h *PacketHeap) Pop() any       { n := len(h.packets); x := h.packets[n-1]; h.packets = h.packets[:n-1]; return x }

func effective(pkt bus.CognitivePacket, now int64) int {
	age := float64(now - pkt.EnqueuedAt)
	return pkt.EffectivePriority + int(math.Log1p(age)*5) + pkt.FairnessBoost
}

type Scheduler struct {
	clock         Clock
	tick          time.Duration
	inbox         chan bus.CognitivePacket
	heap          *PacketHeap
	policy        RoutingPolicy
	now           int64
	mode          SchedulerMode
	mu            sync.Mutex
	handlers      map[bus.PacketType][]func(bus.CognitivePacket)
	output        []bus.CognitivePacket
	resolver      PriorityResolver
	fairness      FairnessController
	droppedCount  atomic.Int64
	processedHigh atomic.Int64
}

func NewScheduler(tickMs int, clock Clock) *Scheduler {
	s := &Scheduler{
		clock: clock, tick: time.Duration(tickMs) * time.Millisecond,
		inbox: make(chan bus.CognitivePacket, 4096), heap: &PacketHeap{},
		policy: DefaultRoutingPolicy, handlers: make(map[bus.PacketType][]func(bus.CognitivePacket)),
	}
	s.heap.scheduler = s
	return s
}

func (s *Scheduler) SetResolver(r PriorityResolver)             { s.resolver = r }
func (s *Scheduler) SetFairnessController(fc FairnessController) { s.fairness = fc }
func (s *Scheduler) Dropped() int64                               { return s.droppedCount.Load() }
func (s *Scheduler) Clock() Clock                                 { return s.clock }

func (s *Scheduler) Register(t bus.PacketType, h func(bus.CognitivePacket)) {
	s.mu.Lock()
	s.handlers[t] = append(s.handlers[t], h)
	s.mu.Unlock()
}

func (s *Scheduler) Emit(pkt bus.CognitivePacket) bool {
	pkt.InitialTTL = pkt.TTL
	if s.resolver != nil {
		pkt = s.resolver.Resolve(pkt)
	}
	if ok := true; s.resolver != nil {
		pkt, ok = s.resolver.ShouldRequeue(pkt)
		if !ok {
			s.droppedCount.Add(1)
			return false
		}
	}
	pkt.EnqueuedAt = s.clock.NowMilli()
	select {
	case s.inbox <- pkt:
		return true
	default:
		s.droppedCount.Add(1)
		return false
	}
}

func (s *Scheduler) Tick() {
	s.now = s.clock.NowMilli()
	s.drainInbox()
	if s.fairness != nil {
		s.fairness.ApplyFairness(s.heap.packets, s.now)
	}
	heap.Init(s.heap)
	s.dispatch()
}

func (s *Scheduler) drainInbox() {
	for {
		select {
		case pkt := <-s.inbox:
			heap.Push(s.heap, pkt)
		default:
			return
		}
	}
}

func (s *Scheduler) dispatch() {
	s.mu.Lock()
	for s.heap.Len() > 0 {
		pkt := heap.Pop(s.heap).(bus.CognitivePacket)
		if pkt.TTL <= 0 {
			continue
		}
		pkt.TTL--
		if pkt.Type == bus.Output {
			s.output = append(s.output, pkt)
		}
		if pkt.EffectivePriority >= 70 {
			s.processedHigh.Add(1)
		}
		handlers := make([]func(bus.CognitivePacket), len(s.handlers[pkt.Type]))
		copy(handlers, s.handlers[pkt.Type])
		s.mu.Unlock()
		for _, h := range handlers {
			h(pkt)
		}
		s.mu.Lock()
	}
	s.mu.Unlock()
}

func (s *Scheduler) Drain() []bus.CognitivePacket {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := s.output
	s.output = nil
	return o
}

func (s *Scheduler) FlushOutput() []bus.CognitivePacket {
	s.mu.Lock()
	o := s.output
	s.output = nil
	s.mu.Unlock()
	return o
}

func (s *Scheduler) QueueSize() int            { return len(s.inbox) + s.heap.Len() }
func (s *Scheduler) SetMode(mode SchedulerMode) { s.mode = mode }
