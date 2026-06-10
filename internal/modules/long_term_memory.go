package modules

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/memory"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type EpisodicRecord struct {
	ID, Context, Details                           string
	StateFootprint, Valencia, Intensidad, Saturacion, Significance float64
	Timestamp                                                       int64
}

type LTMSemanticNode struct {
	NodeID, NodeData string
	Edges            []string
	Strength         float64
}

type LongTermMemory struct {
	sched         *scheduler.Scheduler
	clock         scheduler.Clock
	stateReg      *StateRegister
	mem           *memory.MemoryBus
	episodicLog   []EpisodicRecord
	semanticGraph map[string]*LTMSemanticNode
	mu            sync.RWMutex
}

func NewLongTermMemory(stateReg *StateRegister, mem *memory.MemoryBus, clock scheduler.Clock) *LongTermMemory {
	return &LongTermMemory{
		stateReg: stateReg, mem: mem, clock: clock,
		episodicLog:   make([]EpisodicRecord, 0, 1000),
		semanticGraph: make(map[string]*LTMSemanticNode),
	}
}

func (ltm *LongTermMemory) SetScheduler(s *scheduler.Scheduler) { ltm.sched = s }

func (ltm *LongTermMemory) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Action: ltm.archiveDecision(pkt)
	case bus.Meta: ltm.archiveMeta(pkt)
	case bus.Memory: ltm.handleMemoryQuery(pkt)
	}
}

func (ltm *LongTermMemory) archiveDecision(pkt bus.CognitivePacket) {
	state := ltm.stateReg.GetState()
	var payload map[string]interface{}
	json.Unmarshal(pkt.Payload, &payload)
	decision, _ := payload["decision"].(string)
	message, _ := payload["message"].(string)
	significance := state.Intensidad * 0.4
	if decision == "inhibit" { significance += 0.3 }
	if state.Stress() > 0.6 { significance += 0.2 }
	if significance < 0.3 { return }
	record := EpisodicRecord{
		ID: fmt.Sprintf("ep_%d", ltm.clock.NowMilli()), Context: fmt.Sprintf("decision:%s", decision),
		StateFootprint: state.Stress(), Valencia: state.Valencia, Intensidad: state.Intensidad,
		Saturacion: state.Saturacion, Details: message, Timestamp: ltm.clock.NowMilli(),
		Significance: significance,
	}
	ltm.mu.Lock()
	ltm.episodicLog = append(ltm.episodicLog, record)
	if len(ltm.episodicLog) > 1000 { ltm.episodicLog = ltm.episodicLog[1:] }
	ltm.mu.Unlock()
	recordJSON, _ := json.Marshal(record)
	ltm.mem.Write(record.ID, recordJSON, ltm.clock.NowMilli())
}

func (ltm *LongTermMemory) archiveMeta(pkt bus.CognitivePacket) {}

func (ltm *LongTermMemory) handleMemoryQuery(pkt bus.CognitivePacket) {
	key := string(pkt.Payload)
	value, found := ltm.mem.Read(key)
	if found {
		ltm.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("ltm_%s", pkt.ID), Type: bus.Memory, Source: "long_term_memory",
			Target: pkt.Source, Priority: 55, Timestamp: ltm.clock.NowMilli(),
			Payload: value, TTL: 3,
		})
	}
}

func (ltm *LongTermMemory) PeriodicArchive() {
	state := ltm.stateReg.GetState()
	record := EpisodicRecord{
		ID: fmt.Sprintf("per_%d", ltm.clock.NowMilli()), Context: "periodic_snapshot",
		StateFootprint: state.Stress(), Valencia: state.Valencia, Intensidad: state.Intensidad,
		Saturacion: state.Saturacion, Details: fmt.Sprintf("sat=%.2f int=%.2f", state.Saturacion, state.Intensidad),
		Timestamp: ltm.clock.NowMilli(), Significance: 0.3,
	}
	ltm.mu.Lock()
	ltm.episodicLog = append(ltm.episodicLog, record)
	if len(ltm.episodicLog) > 1000 { ltm.episodicLog = ltm.episodicLog[1:] }
	ltm.mu.Unlock()
	recordJSON, _ := json.Marshal(record)
	ltm.mem.Write(record.ID, recordJSON, ltm.clock.NowMilli())
}

func (ltm *LongTermMemory) RetrieveByEmotionalSignature(pattern string, threshold float64) []EpisodicRecord {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()
	var results []EpisodicRecord
	for _, ep := range ltm.episodicLog {
		if ep.Significance >= threshold && (pattern == "" || containsSubstr(ep.Details, pattern) || containsSubstr(ep.Context, pattern)) {
			results = append(results, ep)
		}
	}
	return results
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || findSubstr(s, sub))
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub { return true }
	}
	return false
}

func (ltm *LongTermMemory) FindAssociations(concept string, limit int) []*LTMSemanticNode {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()
	var results []*LTMSemanticNode
	for _, node := range ltm.semanticGraph {
		if len(results) >= limit { break }
		if node.NodeID == concept || node.NodeData == concept {
			results = append(results, node)
			continue
		}
		for _, edge := range node.Edges {
			if edge == concept {
				results = append(results, node)
				break
			}
		}
	}
	return results
}

func (ltm *LongTermMemory) LinkConcepts(a, b, relation string, weight float64) {
	ltm.mu.Lock()
	defer ltm.mu.Unlock()
	if ltm.semanticGraph[a] == nil {
		ltm.semanticGraph[a] = &LTMSemanticNode{NodeID: a, NodeData: a, Edges: []string{b}, Strength: weight}
	} else {
		for _, edge := range ltm.semanticGraph[a].Edges {
			if edge == b { return }
		}
		ltm.semanticGraph[a].Edges = append(ltm.semanticGraph[a].Edges, b)
		ltm.semanticGraph[a].Strength = (ltm.semanticGraph[a].Strength + weight) / 2
	}
}
