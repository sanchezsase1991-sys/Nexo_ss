package modules

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// SpatialIntegrator — SECCIÓN 10: Módulo de Integración Espacial
// Responsable de:
// 1. Integración sensorial multimodal
// 2. Mapeo de orientación espacial
// 3. Detección de distancia y proximidad
// 4. Modelado de relaciones espaciales
// 5. Atención espacial sesgada hacia estímulos socialmente relevantes

type SpatialIntegrator struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	mu          sync.RWMutex
	spatialMap  SpatialMap
	sensorCache []SensorInput
	maxCache    int
	stats       SpatialStats
}

type SpatialMap struct {
	Origin    Point3D
	Objects   []SpatialObject
	Agents    []AgentPresence
	Boundaries []Boundary
	LastUpdate int64
}

type SpatialObject struct {
	ID        string
	Position  Point3D
	Size      Point3D
	Relevance float64
	Tags      []string
	Timestamp int64
}

type AgentPresence struct {
	ID         string
	Position   Point3D
	Expression string
	EmotionalState string
	Relevance  float64
	Timestamp  int64
}

type Boundary struct {
	Type   string // wall, door, obstacle
	Points []Point3D
}

type Point3D struct {
	X, Y, Z float64
}

type SensorInput struct {
	Type      string
	Data      map[string]any
	Timestamp int64
}

type SpatialStats struct {
	TotalObjects    int
	TotalAgents     int
	AvgRelevance    float64
	SocialBias      float64
	UpdateFrequency float64
}

func NewSpatialIntegrator(stateReg *StateRegister, clock scheduler.Clock) *SpatialIntegrator {
	return &SpatialIntegrator{
		stateReg:    stateReg,
		clock:       clock,
		spatialMap:  SpatialMap{Origin: Point3D{0, 0, 0}},
		sensorCache: make([]SensorInput, 0, 32),
		maxCache:    32,
	}
}

func (si *SpatialIntegrator) SetScheduler(s *scheduler.Scheduler) { si.sched = s }

func (si *SpatialIntegrator) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Perception:
		si.processSensorInput(pkt)
	case bus.Meta:
		si.processMeta(pkt)
	case bus.Thought:
		si.processThoughtForSpatial(pkt)
	}
}

func (si *SpatialIntegrator) processSensorInput(pkt bus.CognitivePacket) {
	var input map[string]any
	json.Unmarshal(pkt.Payload, &input)

	// Crear entrada de sensor
	sensorInput := SensorInput{
		Type:      fmt.Sprintf("%v", input["type"]),
		Data:      input,
		Timestamp: si.clock.NowMilli(),
	}

	// Guardar en caché
	si.mu.Lock()
	si.sensorCache = append(si.sensorCache, sensorInput)
	if len(si.sensorCache) > si.maxCache {
		si.sensorCache = si.sensorCache[1:]
	}
	si.mu.Unlock()

	// Procesar según tipo de sensor
	switch sensorInput.Type {
	case "visual":
		si.processVisualSensor(input, pkt)
	case "audio":
		si.processAudioSensor(input, pkt)
	case "tactile":
		si.processTactileSensor(input, pkt)
	default:
		si.processGeneralSensor(input, pkt)
	}

	// Actualizar mapa espacial
	si.updateSpatialMap()
}

func (si *SpatialIntegrator) processMeta(pkt bus.CognitivePacket) {
	var meta map[string]any
	json.Unmarshal(pkt.Payload, &meta)

	// Obtener mapa espacial
	if getMap, ok := meta["get_map"].(bool); ok && getMap {
		si.emitSpatialMap()
	}

	// Obtener objetos cercanos
	if getNearby, ok := meta["get_nearby"].(map[string]any); ok {
		si.emitNearbyObjects(getNearby)
	}

	// Obtener agentes cercanos
	if getAgents, ok := meta["get_agents"].(bool); ok && getAgents {
		si.emitNearbyAgents()
	}

	// Estadísticas
	if statsReq, ok := meta["request_stats"].(bool); ok && statsReq {
		stats := si.GetStats()
		payload, _ := json.Marshal(stats)
		si.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("spatial_stats_%d", si.clock.NowMilli()), Type: bus.Meta,
			Source: "spatial_integrator", Target: "meta_cognition",
			Priority: 40, Timestamp: si.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

func (si *SpatialIntegrator) processThoughtForSpatial(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil {
		return
	}

	// Buscar contexto espacial relevante
	if thought.IsSocial() {
		// Buscar agentes cercanos
		agents := si.findRelevantAgents(thought)
		if len(agents) > 0 {
			si.emitSpatialRecall(agents, thought)
		}
	}
}

func (si *SpatialIntegrator) processVisualSensor(input map[string]any, pkt bus.CognitivePacket) {
	// Procesar entrada visual para información espacial
	if position, ok := input["position"].(map[string]float64); ok {
		obj := SpatialObject{
			ID: fmt.Sprintf("vis_%s", pkt.ID),
			Position: Point3D{
				X: position["x"],
				Y: position["y"],
				Z: 0,
			},
			Relevance: 0.5,
			Timestamp: si.clock.NowMilli(),
		}

		si.mu.Lock()
		si.spatialMap.Objects = append(si.spatialMap.Objects, obj)
		si.mu.Unlock()
	}

	// Detectar agentes
	if agent, ok := input["agent"].(map[string]any); ok {
		si.processAgentDetection(agent, pkt)
	}
}

func (si *SpatialIntegrator) processAudioSensor(input map[string]any, pkt bus.CognitivePacket) {
	// Procesar entrada de audio para información espacial
	if position, ok := input["source_position"].(map[string]float64); ok {
		// La dirección del sonido indica posición
		obj := SpatialObject{
			ID: fmt.Sprintf("audio_%s", pkt.ID),
			Position: Point3D{
				X: position["x"],
				Y: position["y"],
				Z: 0,
			},
			Relevance: 0.4,
			Tags:      []string{"audio_source"},
			Timestamp: si.clock.NowMilli(),
		}

		si.mu.Lock()
		si.spatialMap.Objects = append(si.spatialMap.Objects, obj)
		si.mu.Unlock()
	}
}

func (si *SpatialIntegrator) processTactileSensor(input map[string]any, pkt bus.CognitivePacket) {
	// Procesar entrada táctil para información de proximidad
	if distance, ok := input["distance"].(float64); ok {
		// La distancia táctil indica proximidad
		si.updateProximityEstimate(distance)
	}
}

func (si *SpatialIntegrator) processGeneralSensor(input map[string]any, pkt bus.CognitivePacket) {
	// Procesamiento general de sensores
	if position, ok := input["position"].(map[string]float64); ok {
		obj := SpatialObject{
			ID: fmt.Sprintf("gen_%s", pkt.ID),
			Position: Point3D{
				X: position["x"],
				Y: position["y"],
				Z: 0,
			},
			Relevance: 0.3,
			Timestamp: si.clock.NowMilli(),
		}

		si.mu.Lock()
		si.spatialMap.Objects = append(si.spatialMap.Objects, obj)
		si.mu.Unlock()
	}
}

func (si *SpatialIntegrator) processAgentDetection(agent map[string]any, pkt bus.CognitivePacket) {
	presence := AgentPresence{
		ID: fmt.Sprintf("agent_%s", pkt.ID),
		Position: Point3D{X: 0, Y: 0, Z: 0},
		Relevance: 0.7,
		Timestamp: si.clock.NowMilli(),
	}

	if position, ok := agent["position"].(map[string]float64); ok {
		presence.Position = Point3D{
			X: position["x"],
			Y: position["y"],
			Z: 0,
		}
	}

	if expr, ok := agent["expression"].(string); ok {
		presence.Expression = expr
	}

	if emotion, ok := agent["emotional_state"].(string); ok {
		presence.EmotionalState = emotion
	}

	si.mu.Lock()
	si.spatialMap.Agents = append(si.spatialMap.Agents, presence)
	si.mu.Unlock()
}

func (si *SpatialIntegrator) updateSpatialMap() {
	si.mu.Lock()
	defer si.mu.Unlock()

	// Limpiar objetos antiguos
	now := si.clock.NowMilli()
	validObjects := make([]SpatialObject, 0)
	for _, obj := range si.spatialMap.Objects {
		if now-obj.Timestamp < 5000 { // 5 segundos
			validObjects = append(validObjects, obj)
		}
	}
	si.spatialMap.Objects = validObjects

	// Limpiar agentes antiguos
	validAgents := make([]AgentPresence, 0)
	for _, agent := range si.spatialMap.Agents {
		if now-agent.Timestamp < 3000 { // 3 segundos
			validAgents = append(validAgents, agent)
		}
	}
	si.spatialMap.Agents = validAgents

	si.spatialMap.LastUpdate = now
}

func (si *SpatialIntegrator) updateProximityEstimate(distance float64) {
	si.mu.Lock()
	defer si.mu.Unlock()

	// Actualizar mapa basado en proximidad
	si.stats.SocialBias = distance * 0.1
}

func (si *SpatialIntegrator) findRelevantAgents(thought bus.ThoughtState) []AgentPresence {
	si.mu.RLock()
	defer si.mu.RUnlock()

	var relevant []AgentPresence
	for _, agent := range si.spatialMap.Agents {
		if agent.Relevance > 0.5 {
			relevant = append(relevant, agent)
		}
	}
	return relevant
}

func (si *SpatialIntegrator) emitSpatialMap() {
	si.mu.RLock()
	defer si.mu.RUnlock()

	payload, _ := json.Marshal(si.spatialMap)
	si.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("spatial_map_%d", si.clock.NowMilli()), Type: bus.Meta,
		Source: "spatial_integrator", Target: "meta_cognition",
		Priority: 40, Timestamp: si.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (si *SpatialIntegrator) emitNearbyObjects(params map[string]any) {
	si.mu.RLock()
	defer si.mu.RUnlock()

	radius := 1.0 // Default radius
	if r, ok := params["radius"].(float64); ok {
		radius = r
	}

	var nearby []SpatialObject
	for _, obj := range si.spatialMap.Objects {
		dist := si.calculateDistance(si.spatialMap.Origin, obj.Position)
		if dist <= radius {
			nearby = append(nearby, obj)
		}
	}

	payload, _ := json.Marshal(nearby)
	si.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("nearby_%d", si.clock.NowMilli()), Type: bus.Meta,
		Source: "spatial_integrator", Target: "perception_gate",
		Priority: 60, Timestamp: si.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (si *SpatialIntegrator) emitNearbyAgents() {
	si.mu.RLock()
	defer si.mu.RUnlock()

	payload, _ := json.Marshal(si.spatialMap.Agents)
	si.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("agents_%d", si.clock.NowMilli()), Type: bus.Meta,
		Source: "spatial_integrator", Target: "social_context_analyzer",
		Priority: 70, Timestamp: si.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (si *SpatialIntegrator) emitSpatialRecall(agents []AgentPresence, thought bus.ThoughtState) {
	payload, _ := json.Marshal(map[string]any{
		"recall_type": "spatial",
		"agents":      agents,
		"query":       thought.Payload,
	})

	si.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("srecall_%s", thought.OriginalID), Type: bus.Memory,
		Source: "spatial_integrator", Target: "long_term_memory",
		Priority: 60, Timestamp: si.clock.NowMilli(),
		Payload: payload, Tags: []string{"spatial_recall"}, TTL: 3,
	})
}

func (si *SpatialIntegrator) calculateDistance(a, b Point3D) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (si *SpatialIntegrator) GetStats() SpatialStats {
	si.mu.Lock()
	defer si.mu.Unlock()

	stats := si.stats
	stats.TotalObjects = len(si.spatialMap.Objects)
	stats.TotalAgents = len(si.spatialMap.Agents)

	if stats.TotalObjects > 0 {
		totalRelevance := 0.0
		for _, obj := range si.spatialMap.Objects {
			totalRelevance += obj.Relevance
		}
		stats.AvgRelevance = totalRelevance / float64(stats.TotalObjects)
	}

	return stats
}

func (si *SpatialIntegrator) GetSpatialMap() SpatialMap {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return si.spatialMap
}
