package modules

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// KnowledgeLinker vincula información nueva con conocimiento previo
type KnowledgeLinker struct {
	sched      *scheduler.Scheduler
	clock      scheduler.Clock
	ltm        *LongTermMemory
	mu         sync.Mutex
	linkCount  int
	unintegrated []string
}

func NewKnowledgeLinker(ltm *LongTermMemory, clock scheduler.Clock) *KnowledgeLinker {
	return &KnowledgeLinker{
		ltm:          ltm,
		clock:        clock,
		unintegrated: make([]string, 0),
	}
}

func (kl *KnowledgeLinker) SetScheduler(s *scheduler.Scheduler) { kl.sched = s }

// Handle procesa pensamientos para vincular con conocimiento previo
func (kl *KnowledgeLinker) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }

	thought, err := pkt.AsThoughtState()
	if err != nil { return }

	// Buscar conexiones con la red semántica
	connections := kl.findSemanticConnections(thought)

	if len(connections) > 0 {
		kl.establishLinks(thought, connections)
	} else {
		kl.markUnintegrated(thought)
	}
}

func (kl *KnowledgeLinker) findSemanticConnections(thought bus.ThoughtState) []string {
	if kl.ltm == nil { return nil }

	var connections []string

	// Buscar en la red semántica por similitud, contraste, analogía
	keywords := extractKeywords(thought.Payload)
	for _, kw := range keywords {
		nodes := kl.ltm.FindAssociations(kw, 2)
		for _, node := range nodes {
			if !containsStr(connections, node.NodeID) {
				connections = append(connections, node.NodeID)
			}
		}
	}

	return connections
}

func (kl *KnowledgeLinker) establishLinks(thought bus.ThoughtState, connections []string) {
	kl.mu.Lock()
	kl.linkCount++
	count := kl.linkCount
	kl.mu.Unlock()

	// Crear nuevas aristas en la red semántica
	for _, conn := range connections {
		kl.ltm.LinkConcepts(thought.OriginalID, conn, "semantic_association", 0.6)
	}

	// Emitir confirmación de vinculación
	payload, _ := json.Marshal(map[string]interface{}{
		"thought_id":  thought.OriginalID,
		"connections": connections,
		"link_count":  count,
	})

	kl.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("linked_%s", thought.OriginalID),
		Type:      bus.Meta,
		Source:    "knowledge_linker",
		Target:    "long_term_memory",
		Priority:  45,
		Timestamp: kl.clock.NowMilli(),
		Payload:   payload,
		Tags:      []string{"knowledge_link", "semantic_connection"},
		TTL:       3,
	})
}

func (kl *KnowledgeLinker) markUnintegrated(thought bus.ThoughtState) {
	kl.mu.Lock()
	kl.unintegrated = append(kl.unintegrated, thought.OriginalID)
	if len(kl.unintegrated) > 50 { kl.unintegrated = kl.unintegrated[1:] }
	unintegratedCount := len(kl.unintegrated)
	kl.mu.Unlock()

	// Emitir flag de "no integrado"
	kl.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("unintegrated_%s", thought.OriginalID),
		Type:      bus.Meta,
		Source:    "knowledge_linker",
		Target:    "state_register",
		Priority:  25,
		Timestamp: kl.clock.NowMilli(),
		Payload:   []byte(fmt.Sprintf(`{"unintegrated_count":%d,"flag":"knowledge_gap"}`, unintegratedCount)),
		Tags:      []string{"unintegrated", "knowledge_gap"},
		TTL:       5,
	})
}

func (kl *KnowledgeLinker) GetUnintegratedCount() int {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	return len(kl.unintegrated)
}

func extractKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	if len(words) > 5 { return words[:5] }
	return words
}
