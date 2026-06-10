package modules

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// VisualProcessor — SECCIÓN 8: Módulo de Procesamiento Visual
// Responsable de:
// 1. Procesamiento de movimiento y reconocimiento facial
// 2. Codificación de características visuales
// 3. Coordinación oculomotora
// 4. Integración visual-táctil
// 5. Detección de cambios sutiles en expresiones

type VisualProcessor struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	mu          sync.RWMutex
	faceCache   []FaceDetection
	objectCache []VisualObject
	maxCache    int
	stats       VisualStats
}

type FaceDetection struct {
	ID           string
	Expression    string
	Confidence   float64
	EmotionMap   map[string]float64
	Position     Point2D
	Timestamp    int64
}

type VisualObject struct {
	ID         string
	Type       string
	Position   Point2D
	Size       Point2D
	Movement   Vector2D
	Relevance  float64
	Timestamp  int64
}

type Point2D struct {
	X, Y float64
}

type Vector2D struct {
	DX, DY float64
}

type VisualStats struct {
	TotalDetections   int
	FaceDetections    int
	ObjectDetections  int
	AvgConfidence     float64
	ExpressionChanges int
}

func NewVisualProcessor(stateReg *StateRegister, clock scheduler.Clock) *VisualProcessor {
	return &VisualProcessor{
		stateReg:    stateReg,
		clock:       clock,
		faceCache:   make([]FaceDetection, 0, 32),
		objectCache: make([]VisualObject, 0, 64),
		maxCache:    64,
	}
}

func (vp *VisualProcessor) SetScheduler(s *scheduler.Scheduler) { vp.sched = s }

func (vp *VisualProcessor) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Perception:
		vp.processVisualInput(pkt)
	case bus.Meta:
		vp.processMeta(pkt)
	case bus.Thought:
		vp.processThoughtForVisual(pkt)
	}
}

func (vp *VisualProcessor) processVisualInput(pkt bus.CognitivePacket) {
	var input map[string]any
	json.Unmarshal(pkt.Payload, &input)

	// Procesar según tipo de entrada visual
	inputType, _ := input["type"].(string)

	switch inputType {
	case "face":
		vp.processFaceDetection(input, pkt)
	case "object":
		vp.processObjectDetection(input, pkt)
	case "movement":
		vp.processMovementDetection(input, pkt)
	default:
		vp.processGeneralVisual(input, pkt)
	}
}

func (vp *VisualProcessor) processFaceDetection(input map[string]any, pkt bus.CognitivePacket) {
	detection := FaceDetection{
		ID:         fmt.Sprintf("face_%s", pkt.ID),
		Timestamp:  vp.clock.NowMilli(),
		EmotionMap: make(map[string]float64),
	}

	if expr, ok := input["expression"].(string); ok {
		detection.Expression = expr
	}
	if conf, ok := input["confidence"].(float64); ok {
		detection.Confidence = conf
	}
	if emotions, ok := input["emotions"].(map[string]float64); ok {
		detection.EmotionMap = emotions
	}

	vp.mu.Lock()
	vp.faceCache = append(vp.faceCache, detection)
	if len(vp.faceCache) > vp.maxCache {
		vp.faceCache = vp.faceCache[1:]
	}
	vp.stats.FaceDetections++
	vp.stats.TotalDetections++
	vp.mu.Unlock()

	// Analizar cambios de expresión
	vp.analyzeExpressionChange(detection)

	// Emitir resultado
	vp.emitFaceAnalysis(detection, pkt)
}

func (vp *VisualProcessor) processObjectDetection(input map[string]any, pkt bus.CognitivePacket) {
	object := VisualObject{
		ID:        fmt.Sprintf("obj_%s", pkt.ID),
		Timestamp: vp.clock.NowMilli(),
	}

	if objType, ok := input["object_type"].(string); ok {
		object.Type = objType
	}
	if pos, ok := input["position"].(map[string]float64); ok {
		object.Position = Point2D{X: pos["x"], Y: pos["y"]}
	}
	if size, ok := input["size"].(map[string]float64); ok {
		object.Size = Point2D{X: size["width"], Y: size["height"]}
	}
	if movement, ok := input["movement"].(map[string]float64); ok {
		object.Movement = Vector2D{DX: movement["dx"], DY: movement["dy"]}
	}

	vp.mu.Lock()
	vp.objectCache = append(vp.objectCache, object)
	if len(vp.objectCache) > vp.maxCache {
		vp.objectCache = vp.objectCache[1:]
	}
	vp.stats.ObjectDetections++
	vp.stats.TotalDetections++
	vp.mu.Unlock()

	// Emitir resultado
	vp.emitObjectAnalysis(object, pkt)
}

func (vp *VisualProcessor) processMovementDetection(input map[string]any, pkt bus.CognitivePacket) {
	// Detectar movimiento significativo
	speed, _ := input["speed"].(float64)
	direction, _ := input["direction"].(float64)

	// El movimiento es relevante si es rápido o en dirección al sistema
	if speed > 0.5 || (direction > 0.3 && direction < 0.7) {
		vp.emitMovementAlert(speed, direction, pkt)
	}
}

func (vp *VisualProcessor) processGeneralVisual(input map[string]any, pkt bus.CognitivePacket) {
	// Procesamiento visual general
	relevance := 0.0
	if rel, ok := input["relevance"].(float64); ok {
		relevance = rel
	}

	// Solo emitir si es relevante
	if relevance > 0.3 {
		vp.emitGeneralVisualAlert(input, relevance, pkt)
	}
}

func (vp *VisualProcessor) processMeta(pkt bus.CognitivePacket) {
	var meta map[string]any
	json.Unmarshal(pkt.Payload, &meta)

	// Obtener caché de caras
	if getFaces, ok := meta["get_faces"].(bool); ok && getFaces {
		vp.emitFaceCache()
	}

	// Obtener caché de objetos
	if getObjects, ok := meta["get_objects"].(bool); ok && getObjects {
		vp.emitObjectCache()
	}

	// Estadísticas
	if statsReq, ok := meta["request_stats"].(bool); ok && statsReq {
		stats := vp.GetStats()
		payload, _ := json.Marshal(stats)
		vp.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("vis_stats_%d", vp.clock.NowMilli()), Type: bus.Meta,
			Source: "visual_processor", Target: "meta_cognition",
			Priority: 40, Timestamp: vp.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

func (vp *VisualProcessor) processThoughtForVisual(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil {
		return
	}

	// Si el pensamiento contiene información visual
	if thought.HasRichAssociations() {
		// Buscar en caché de caras
		faces := vp.searchFaceCache(thought.Payload)
		if len(faces) > 0 {
			vp.emitVisualRecall(faces, thought)
		}
	}
}

func (vp *VisualProcessor) analyzeExpressionChange(detection FaceDetection) {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if len(vp.faceCache) < 2 {
		return
	}

	lastFace := vp.faceCache[len(vp.faceCache)-2]
	if lastFace.Expression != detection.Expression {
		vp.stats.ExpressionChanges++

		// Emitir alerta de cambio de expresión
		vp.emitExpressionChangeAlert(lastFace, detection)
	}
}

func (vp *VisualProcessor) emitFaceAnalysis(detection FaceDetection, pkt bus.CognitivePacket) {
	payload, _ := json.Marshal(map[string]any{
		"detection_id": detection.ID,
		"expression":   detection.Expression,
		"confidence":   detection.Confidence,
		"emotions":     detection.EmotionMap,
		"position":     detection.Position,
	})

	vp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("face_%s", detection.ID), Type: bus.Thought,
		Source: "visual_processor", Target: "social_context_analyzer",
		Priority: 70, Timestamp: vp.clock.NowMilli(),
		Payload: payload, Tags: []string{"face_detection"}, TTL: 3,
	})
}

func (vp *VisualProcessor) emitObjectAnalysis(object VisualObject, pkt bus.CognitivePacket) {
	payload, _ := json.Marshal(map[string]any{
		"object_id": object.ID,
		"type":      object.Type,
		"position":  object.Position,
		"movement":  object.Movement,
	})

	vp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("obj_%s", object.ID), Type: bus.Thought,
		Source: "visual_processor", Target: "perception_gate",
		Priority: 60, Timestamp: vp.clock.NowMilli(),
		Payload: payload, Tags: []string{"object_detection"}, TTL: 3,
	})
}

func (vp *VisualProcessor) emitMovementAlert(speed, direction float64, pkt bus.CognitivePacket) {
	payload, _ := json.Marshal(map[string]any{
		"alert_type": "movement",
		"speed":      speed,
		"direction":  direction,
	})

	vp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("move_%s", pkt.ID), Type: bus.Meta,
		Source: "visual_processor", Target: "attention_controller",
		Priority: 75, Timestamp: vp.clock.NowMilli(),
		Payload: payload, Tags: []string{"movement_alert"}, TTL: 2,
	})
}

func (vp *VisualProcessor) emitGeneralVisualAlert(input map[string]any, relevance float64, pkt bus.CognitivePacket) {
	input["alert_type"] = "general_visual"
	input["relevance"] = relevance

	payload, _ := json.Marshal(input)
	vp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("vis_%s", pkt.ID), Type: bus.Meta,
		Source: "visual_processor", Target: "perception_gate",
		Priority: int(relevance*100), Timestamp: vp.clock.NowMilli(),
		Payload: payload, Tags: []string{"visual_alert"}, TTL: 2,
	})
}

func (vp *VisualProcessor) emitExpressionChangeAlert(old, new FaceDetection) {
	payload, _ := json.Marshal(map[string]any{
		"alert_type": "expression_change",
		"old":        old.Expression,
		"new":        new.Expression,
		"confidence": new.Confidence,
	})

	vp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("expr_%d", vp.clock.NowMilli()), Type: bus.Meta,
		Source: "visual_processor", Target: "social_context_analyzer",
		Priority: 80, Timestamp: vp.clock.NowMilli(),
		Payload: payload, Tags: []string{"expression_change"}, TTL: 3,
	})
}

func (vp *VisualProcessor) emitFaceCache() {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	payload, _ := json.Marshal(vp.faceCache)
	vp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("face_cache_%d", vp.clock.NowMilli()), Type: bus.Meta,
		Source: "visual_processor", Target: "meta_cognition",
		Priority: 40, Timestamp: vp.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (vp *VisualProcessor) emitObjectCache() {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	payload, _ := json.Marshal(vp.objectCache)
	vp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("obj_cache_%d", vp.clock.NowMilli()), Type: bus.Meta,
		Source: "visual_processor", Target: "meta_cognition",
		Priority: 40, Timestamp: vp.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (vp *VisualProcessor) searchFaceCache(query string) []FaceDetection {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	var results []FaceDetection
	for _, face := range vp.faceCache {
		// Búsqueda simple por expresión
		if containsIgnoreCase(face.Expression, query) {
			results = append(results, face)
		}
	}
	return results
}

func (vp *VisualProcessor) emitVisualRecall(faces []FaceDetection, thought bus.ThoughtState) {
	payload, _ := json.Marshal(map[string]any{
		"recall_type": "visual",
		"faces":       faces,
		"query":       thought.Payload,
	})

	vp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("vrecall_%s", thought.OriginalID), Type: bus.Memory,
		Source: "visual_processor", Target: "long_term_memory",
		Priority: 60, Timestamp: vp.clock.NowMilli(),
		Payload: payload, Tags: []string{"visual_recall"}, TTL: 3,
	})
}

func (vp *VisualProcessor) GetStats() VisualStats {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	return vp.stats
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || len(substr) == 0 || 
			(len(s) > 0 && len(substr) > 0 && 
				(s[0:len(substr)] == substr || 
				 containsIgnoreCase(s[1:], substr))))
}
