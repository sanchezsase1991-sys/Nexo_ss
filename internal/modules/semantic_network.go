package modules

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// SemanticNode representa un nodo en la red semántica densa
type SemanticNode struct {
	NodeID     string            `json:"node_id"`
	Concept    string            `json:"concept"`
	Properties map[string]float64 `json:"properties"`
	Edges      []SemanticEdge    `json:"edges"`
	Strength   float64           `json:"strength"`
	Flags      []string          `json:"flags"`
}

// SemanticEdge representa una arista entre conceptos
type SemanticEdge struct {
	TargetID     string  `json:"target_id"`
	RelationType string  `json:"relation_type"` // similarity, contrast, analogy, causal, metaphor
	Weight       float64 `json:"weight"`
}

// SemanticNetwork implementa la red semántica densa
type SemanticNetwork struct {
	sched      *scheduler.Scheduler
	clock      scheduler.Clock
	nodes      map[string]*SemanticNode
	mu         sync.RWMutex
	totalEdges int
}

func NewSemanticNetwork(clock scheduler.Clock) *SemanticNetwork {
	return &SemanticNetwork{
		clock: clock,
		nodes: make(map[string]*SemanticNode),
	}
}

func (sn *SemanticNetwork) SetScheduler(s *scheduler.Scheduler) { sn.sched = s }

// Handle procesa pensamientos para expandir la red semántica
func (sn *SemanticNetwork) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }

	thought, err := pkt.AsThoughtState()
	if err != nil { return }

	// Extraer conceptos y crear/actualizar nodos
	concepts := sn.extractConcepts(thought.Payload)
	for _, concept := range concepts {
		sn.addOrUpdateNode(concept, thought)
	}

	// Crear aristas entre conceptos co-ocurrentes
	if len(concepts) >= 2 {
		sn.createEdges(concepts, thought)
	}
}

func (sn *SemanticNetwork) extractConcepts(payload string) []string {
	words := strings.Fields(strings.ToLower(payload))
	concepts := make([]string, 0)
	for _, w := range words {
		if len(w) > 3 {
			concepts = append(concepts, w)
		}
	}
	return concepts
}

func (sn *SemanticNetwork) addOrUpdateNode(concept string, thought bus.ThoughtState) {
	sn.mu.Lock()
	defer sn.mu.Unlock()

	node, exists := sn.nodes[concept]
	if !exists {
		node = &SemanticNode{
			NodeID:     fmt.Sprintf("node_%d", len(sn.nodes)),
			Concept:    concept,
			Properties: make(map[string]float64),
			Edges:      make([]SemanticEdge, 0),
			Strength:   0.5,
			Flags:      []string{"new"},
		}
		sn.nodes[concept] = node
	}

	node.Properties["intensity"] = thought.Intensity
	node.Properties["relevance"] = thought.Score
	node.Strength = math.Min(node.Strength+0.05, 1.0)
	node.Flags = removeFlag(node.Flags, "new")
}

func (sn *SemanticNetwork) createEdges(concepts []string, thought bus.ThoughtState) {
	sn.mu.Lock()
	defer sn.mu.Unlock()

	for i := 0; i < len(concepts); i++ {
		for j := i + 1; j < len(concepts); j++ {
			source := sn.nodes[concepts[i]]
			target := sn.nodes[concepts[j]]
			if source == nil || target == nil { continue }

			// Verificar si la arista ya existe
			if sn.edgeExists(source, target.NodeID) { continue }

			// Determinar tipo de relación
			relationType := sn.inferRelationType(concepts[i], concepts[j], thought)

			// Crear arista
			edge := SemanticEdge{
				TargetID:     target.NodeID,
				RelationType: relationType,
				Weight:       0.5 + thought.Intensity*0.3,
			}
			source.Edges = append(source.Edges, edge)
			sn.totalEdges++

			// Arista inversa
			reverseEdge := SemanticEdge{
				TargetID:     source.NodeID,
				RelationType: relationType,
				Weight:       edge.Weight,
			}
			target.Edges = append(target.Edges, reverseEdge)
		}
	}
}

func (sn *SemanticNetwork) inferRelationType(a, b string, thought bus.ThoughtState) string {
	if thought.IsSocial() { return "social_association" }
	if thought.IsUrgent() { return "causal" }
	if thought.IsQuestion() { return "analogy" }
	return "similarity"
}

func (sn *SemanticNetwork) edgeExists(node *SemanticNode, targetID string) bool {
	for _, e := range node.Edges {
		if e.TargetID == targetID { return true }
	}
	return false
}

// PropagateActivation propaga activación desde un concepto
func (sn *SemanticNetwork) PropagateActivation(concept string, initialActivation float64, depth int) map[string]float64 {
	sn.mu.RLock()
	defer sn.mu.RUnlock()

	activated := make(map[string]float64)
	activated[concept] = initialActivation
	currentLevel := []string{concept}

	for d := 0; d < depth && len(currentLevel) > 0; d++ {
		var nextLevel []string
		for _, nodeID := range currentLevel {
			node, exists := sn.nodes[nodeID]
			if !exists { continue }

			currentActivation := activated[nodeID]
			for _, edge := range node.Edges {
				propagated := currentActivation * edge.Weight * 0.6
				if existing, ok := activated[edge.TargetID]; !ok || propagated > existing {
					activated[edge.TargetID] = propagated
					nextLevel = append(nextLevel, edge.TargetID)
				}
			}
		}
		currentLevel = nextLevel
	}

	return activated
}

// GetNode retorna un nodo por concepto
func (sn *SemanticNetwork) GetNode(concept string) *SemanticNode {
	sn.mu.RLock()
	defer sn.mu.RUnlock()
	return sn.nodes[concept]
}

// GetStats retorna estadísticas de la red
func (sn *SemanticNetwork) GetStats() map[string]interface{} {
	sn.mu.RLock()
	defer sn.mu.RUnlock()
	return map[string]interface{}{
		"total_nodes": len(sn.nodes),
		"total_edges": sn.totalEdges,
		"avg_strength": sn.calculateAvgStrength(),
	}
}

func (sn *SemanticNetwork) calculateAvgStrength() float64 {
	if len(sn.nodes) == 0 { return 0 }
	sum := 0.0
	for _, node := range sn.nodes { sum += node.Strength }
	return sum / float64(len(sn.nodes))
}

func removeFlag(flags []string, flag string) []string {
	var result []string
	for _, f := range flags {
		if f != flag { result = append(result, f) }
	}
	return result
}
