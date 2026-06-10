package modules

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// ProsodyAnalyzer — SECCIÓN 9: Módulo de Análisis de Prosodia y Tono
// Responsable de:
// 1. Análisis de contorno de tono
// 2. Marcadores de postura y estado emocional
// 3. Percepción de emoción a través del tono
// 4. Equilibrio entre tono y contenido
// 5. Detección de incongruencias tono-contenido

type ProsodyAnalyzer struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	mu          sync.RWMutex
	prosodyHistory []ProsodySample
	maxHistory  int
	stats       ProsodyStats
}

type ProsodySample struct {
	ID          string
	ToneContour []float64 // Puntos del contorno de tono
	Speed       float64   // Velocidad del habla (0-1)
	PauseLength float64   // Longitud de pausas (0-1)
	Emphasis    []string  // Palabras enfatizadas
	Emotion     string    // Emoción detectada
	Confidence  float64   // Confianza en la detección
	Timestamp   int64
}

type ProsodyStats struct {
	TotalSamples    int
	AvgSpeed        float64
	AvgPauseLength  float64
	EmotionDistribution map[string]int
	AvgConfidence   float64
	Incongruences   int
}

func NewProsodyAnalyzer(stateReg *StateRegister, clock scheduler.Clock) *ProsodyAnalyzer {
	return &ProsodyAnalyzer{
		stateReg:      stateReg,
		clock:         clock,
		prosodyHistory: make([]ProsodySample, 0, 64),
		maxHistory:    64,
		stats: ProsodyStats{
			EmotionDistribution: make(map[string]int),
		},
	}
}

func (pa *ProsodyAnalyzer) SetScheduler(s *scheduler.Scheduler) { pa.sched = s }

func (pa *ProsodyAnalyzer) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Perception:
		pa.processAudioInput(pkt)
	case bus.Meta:
		pa.processMeta(pkt)
	case bus.Thought:
		pa.processThoughtForProsody(pkt)
	}
}

func (pa *ProsodyAnalyzer) processAudioInput(pkt bus.CognitivePacket) {
	var input map[string]any
	json.Unmarshal(pkt.Payload, &input)

	// Crear muestra de prosodia
	sample := pa.createProsodySample(input, pkt.ID)

	// Analizar contorno de tono
	contour := pa.analyzeToneContour(sample)
	sample.ToneContour = contour

	// Detectar emoción
	emotion, confidence := pa.detectEmotion(sample)
	sample.Emotion = emotion
	sample.Confidence = confidence

	// Detectar incongruencias
	if pa.detectIncongruence(sample, input) {
		pa.stats.Incongruences++
		pa.emitIncongruenceAlert(sample, pkt)
	}

	// Guardar en historial
	pa.recordSample(sample)

	// Emitir análisis
	pa.emitProsodyAnalysis(sample, pkt)
}

func (pa *ProsodyAnalyzer) processMeta(pkt bus.CognitivePacket) {
	var meta map[string]any
	json.Unmarshal(pkt.Payload, &meta)

	// Obtener historial
	if getHistory, ok := meta["get_history"].(bool); ok && getHistory {
		pa.emitProsodyHistory()
	}

	// Analizar patrón de tono
	if analyzePattern, ok := meta["analyze_pattern"].(string); ok {
		pa.analyzeTonePattern(analyzePattern)
	}

	// Estadísticas
	if statsReq, ok := meta["request_stats"].(bool); ok && statsReq {
		stats := pa.GetStats()
		payload, _ := json.Marshal(stats)
		pa.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("pros_stats_%d", pa.clock.NowMilli()), Type: bus.Meta,
			Source: "prosody_analyzer", Target: "meta_cognition",
			Priority: 40, Timestamp: pa.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

func (pa *ProsodyAnalyzer) processThoughtForProsody(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil {
		return
	}

	// Evaluar si el pensamiento contiene información de tono
	if thought.HasRichAssociations() {
		// Buscar en historial de prosodia
		samples := pa.searchProsodyHistory(thought.Payload)
		if len(samples) > 0 {
			pa.emitProsodyRecall(samples, thought)
		}
	}
}

func (pa *ProsodyAnalyzer) createProsodySample(input map[string]any, id string) ProsodySample {
	sample := ProsodySample{
		ID:        fmt.Sprintf("pros_%s", id),
		Emphasis:  make([]string, 0),
		Timestamp: pa.clock.NowMilli(),
	}

	if speed, ok := input["speed"].(float64); ok {
		sample.Speed = clamp(speed, 0, 1)
	}
	if pause, ok := input["pause_length"].(float64); ok {
		sample.PauseLength = clamp(pause, 0, 1)
	}
	if emphasis, ok := input["emphasis"].([]string); ok {
		sample.Emphasis = emphasis
	}

	return sample
}

func (pa *ProsodyAnalyzer) analyzeToneContour(sample ProsodySample) []float64 {
	// Simular análisis de contorno de tono
	// En un sistema real, esto procesaría la señal de audio
	contour := make([]float64, 20)

	// Generar contorno basado en velocidad y pausas
	baseTone := 0.5
	for i := range contour {
		// Variación basada en velocidad
		variation := math.Sin(float64(i)*0.5) * sample.Speed * 0.2
		// Pausas reducen el tono
		if sample.PauseLength > 0.5 {
			variation *= 0.5
		}
		contour[i] = clamp(baseTone+variation, 0, 1)
	}

	return contour
}

func (pa *ProsodyAnalyzer) detectEmotion(sample ProsodySample) (string, float64) {
	// Detección basada en características de prosodia
	emotion := "neutral"
	confidence := 0.5

	// Velocidad alta puede indicar emoción
	if sample.Speed > 0.7 {
		emotion = "excited"
		confidence = 0.7
	} else if sample.Speed < 0.3 {
		emotion = "sad"
		confidence = 0.6
	}

	// Pausas largas pueden indicar tristeza o pensamiento
	if sample.PauseLength > 0.6 {
		emotion = "contemplative"
		confidence = 0.65
	}

	// Énfasis puede indicar enfoque o emoción
	if len(sample.Emphasis) > 2 {
		emotion = "emphatic"
		confidence = 0.6
	}

	return emotion, clamp(confidence, 0, 1)
}

func (pa *ProsodyAnalyzer) detectIncongruence(sample ProsodySample, input map[string]any) bool {
	// Detectar incongruencia entre tono y contenido
	content, _ := input["content"].(string)
	if content == "" {
		return false
	}

	// Verificar si el tono contradice el contenido
	lowerContent := strings.ToLower(content)

	// Contenido positivo con tono triste
	if strings.Contains(lowerContent, "bien") || strings.Contains(lowerContent, "genial") {
		if sample.Emotion == "sad" || sample.Speed < 0.3 {
			return true
		}
	}

	// Contenido negativo con tono alegre
	if strings.Contains(lowerContent, "mal") || strings.Contains(lowerContent, "terrible") {
		if sample.Emotion == "excited" || sample.Speed > 0.7 {
			return true
		}
	}

	return false
}

func (pa *ProsodyAnalyzer) recordSample(sample ProsodySample) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.prosodyHistory = append(pa.prosodyHistory, sample)
	if len(pa.prosodyHistory) > pa.maxHistory {
		pa.prosodyHistory = pa.prosodyHistory[1:]
	}

	// Actualizar estadísticas
	pa.stats.TotalSamples++
	pa.stats.AvgSpeed = sample.Speed*0.1 + pa.stats.AvgSpeed*0.9
	pa.stats.AvgPauseLength = sample.PauseLength*0.1 + pa.stats.AvgPauseLength*0.9
	pa.stats.AvgConfidence = sample.Confidence*0.1 + pa.stats.AvgConfidence*0.9
	pa.stats.EmotionDistribution[sample.Emotion]++
}

func (pa *ProsodyAnalyzer) emitProsodyAnalysis(sample ProsodySample, pkt bus.CognitivePacket) {
	payload, _ := json.Marshal(map[string]any{
		"sample_id":  sample.ID,
		"emotion":    sample.Emotion,
		"confidence": sample.Confidence,
		"speed":      sample.Speed,
		"pauses":     sample.PauseLength,
		"emphasis":   sample.Emphasis,
		"contour":    sample.ToneContour,
	})

	pa.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("pros_%s", sample.ID), Type: bus.Thought,
		Source: "prosody_analyzer", Target: "social_context_analyzer",
		Priority: 70, Timestamp: pa.clock.NowMilli(),
		Payload: payload, Tags: []string{"prosody_analysis"}, TTL: 3,
	})
}

func (pa *ProsodyAnalyzer) emitIncongruenceAlert(sample ProsodySample, pkt bus.CognitivePacket) {
	payload, _ := json.Marshal(map[string]any{
		"alert_type": "prosody_incongruence",
		"emotion":    sample.Emotion,
		"speed":      sample.Speed,
		"pauses":     sample.PauseLength,
	})

	pa.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("incon_%s", pkt.ID), Type: bus.Meta,
		Source: "prosody_analyzer", Target: "inhibitory_control",
		Priority: 85, Timestamp: pa.clock.NowMilli(),
		Payload: payload, Tags: []string{"incongruence_alert"}, TTL: 3,
	})
}

func (pa *ProsodyAnalyzer) emitProsodyHistory() {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	payload, _ := json.Marshal(pa.prosodyHistory)
	pa.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("pros_hist_%d", pa.clock.NowMilli()), Type: bus.Meta,
		Source: "prosody_analyzer", Target: "meta_cognition",
		Priority: 40, Timestamp: pa.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (pa *ProsodyAnalyzer) analyzeTonePattern(pattern string) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	// Buscar patrón en historial
	count := 0
	for _, sample := range pa.prosodyHistory {
		if sample.Emotion == pattern {
			count++
		}
	}

	payload, _ := json.Marshal(map[string]any{
		"pattern": pattern,
		"count":   count,
		"total":   len(pa.prosodyHistory),
	})

	pa.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("pattern_%d", pa.clock.NowMilli()), Type: bus.Meta,
		Source: "prosody_analyzer", Target: "meta_cognition",
		Priority: 50, Timestamp: pa.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (pa *ProsodyAnalyzer) searchProsodyHistory(query string) []ProsodySample {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	var results []ProsodySample
	for _, sample := range pa.prosodyHistory {
		if strings.Contains(sample.Emotion, query) {
			results = append(results, sample)
		}
	}
	return results
}

func (pa *ProsodyAnalyzer) emitProsodyRecall(samples []ProsodySample, thought bus.ThoughtState) {
	payload, _ := json.Marshal(map[string]any{
		"recall_type": "prosody",
		"samples":     samples,
		"query":       thought.Payload,
	})

	pa.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("precall_%s", thought.OriginalID), Type: bus.Memory,
		Source: "prosody_analyzer", Target: "long_term_memory",
		Priority: 60, Timestamp: pa.clock.NowMilli(),
		Payload: payload, Tags: []string{"prosody_recall"}, TTL: 3,
	})
}

func (pa *ProsodyAnalyzer) GetStats() ProsodyStats {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	return pa.stats
}
