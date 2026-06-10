package modules

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// MotivationEngine — SECCIÓN 6: Módulo de Motivación y Emoción
// Responsable de:
// 1. Circuitos de recompensa (propósito, logro, conexión)
// 2. Conteo de struggling (esfuerzo acumulado)
// 3. Motor de propósito (dirección de la motivación)
// 4. Configuración de valencia emocional
// 5. Regulación de intensidad motivacional

type MotivationEngine struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	wm          *WorkingMemoryManager
	mu          sync.RWMutex
	rewards     []RewardSignal
	struggling  float64
	purpose     *PurposeVector
	emotionalState EmotionalState
	stats       MotivationStats
}

type EmotionalState struct {
	Valence    float64 // -1 a 1 (negativo a positivo)
	Arousal    float64 // 0 a 1 (calmado a activado)
	Dominance  float64 // 0 a 1 (sumiso a dominante)
	LastUpdate int64
}

type RewardSignal struct {
	Type      RewardType
	Value     float64
	Source    string
	Timestamp int64
}

type RewardType int

const (
	RewardAchievement RewardType = iota
	RewardSocial
	RewardPurpose
	RewardNovelty
	RewardMastery
)

func (rt RewardType) String() string {
	switch rt {
	case RewardAchievement:
		return "ACHIEVEMENT"
	case RewardSocial:
		return "SOCIAL"
	case RewardPurpose:
		return "PURPOSE"
	case RewardNovelty:
		return "NOVELTY"
	case RewardMastery:
		return "MASTERY"
	default:
		return "UNKNOWN"
	}
}

type PurposeVector struct {
	Direction float64 // Dirección del propósito (0-1)
	Intensity float64 // Intensidad del propósito (0-1)
	Source    string  // Fuente del propósito
	LastSync  int64
}

type MotivationStats struct {
	TotalRewards   int
	RewardRate     float64
	Struggling     float64
	PurposeStrength float64
	EmotionalBal   float64
	AvgArousal     float64
}

func NewMotivationEngine(stateReg *StateRegister, clock scheduler.Clock, wm *WorkingMemoryManager) *MotivationEngine {
	return &MotivationEngine{
		stateReg: stateReg,
		clock:    clock,
		wm:       wm,
		rewards:  make([]RewardSignal, 0, 64),
		purpose:  &PurposeVector{Direction: 0.5, Intensity: 0.3, Source: "internal"},
		emotionalState: EmotionalState{
			Valence:   0.5,
			Arousal:   0.3,
			Dominance: 0.5,
		},
	}
}

func (me *MotivationEngine) SetScheduler(s *scheduler.Scheduler) { me.sched = s }

func (me *MotivationEngine) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Thought:
		me.processThoughtForMotivation(pkt)
	case bus.Action:
		me.processActionResult(pkt)
	case bus.Meta:
		me.processMeta(pkt)
	case bus.Memory:
		me.processMemoryUpdate(pkt)
	}
}

func (me *MotivationEngine) processThoughtForMotivation(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil {
		return
	}

	// Evaluar señal de recompensa
	reward := me.evaluateRewardSignal(thought)
	if reward.Value > 0.3 {
		me.recordReward(reward)
	}

	// Actualizar struggling
	me.updateStruggling(thought)

	// Actualizar propósito
	me.updatePurpose(thought)
}

func (me *MotivationEngine) processActionResult(pkt bus.CognitivePacket) {
	var result map[string]any
	json.Unmarshal(pkt.Payload, &result)

	// Evaluar recompensa por resultado
	if success, ok := result["success"].(bool); ok && success {
		reward := RewardSignal{
			Type:      RewardAchievement,
			Value:     0.4,
			Source:    fmt.Sprintf("action_%v", result["action_id"]),
			Timestamp: me.clock.NowMilli(),
		}
		me.recordReward(reward)
	}
}

func (me *MotivationEngine) processMeta(pkt bus.CognitivePacket) {
	var meta map[string]any
	json.Unmarshal(pkt.Payload, &meta)

	// Añadir recompensa manual
	if rewardReq, ok := meta["add_reward"].(map[string]any); ok {
		reward := me.createRewardFromMap(rewardReq)
		me.recordReward(reward)
	}

	// Actualizar propósito
	if purposeReq, ok := meta["update_purpose"].(map[string]any); ok {
		me.updatePurposeFromMap(purposeReq)
	}

	// Obtener estado emocional
	if getEmotional, ok := meta["get_emotional"].(bool); ok && getEmotional {
		me.emitEmotionalState()
	}

	// Estadísticas
	if statsReq, ok := meta["request_stats"].(bool); ok && statsReq {
		stats := me.GetStats()
		payload, _ := json.Marshal(stats)
		me.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("mot_stats_%d", me.clock.NowMilli()), Type: bus.Meta,
			Source: "motivation_engine", Target: "meta_cognition",
			Priority: 40, Timestamp: me.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

func (me *MotivationEngine) processMemoryUpdate(pkt bus.CognitivePacket) {
	var memEvent map[string]any
	json.Unmarshal(pkt.Payload, &memEvent)

	// Eventos de memoria afectan motivación
	if eventType, ok := memEvent["event"].(string); ok {
		switch eventType {
		case "eviction":
			// Evicción reduce motivación
			me.updateEmotionalState(-0.05, 0.02, 0)
		case "consolidation":
			// Consolidación aumenta motivación
			me.updateEmotionalState(0.03, 0.01, 0.02)
		case "recall_success":
			// Recordar exitosamente aumenta motivación
			me.recordReward(RewardSignal{
				Type:      RewardMastery,
				Value:     0.2,
				Source:    "memory_recall",
				Timestamp: me.clock.NowMilli(),
			})
		}
	}
}

func (me *MotivationEngine) evaluateRewardSignal(thought bus.ThoughtState) RewardSignal {
	value := 0.0
	rewardType := RewardPurpose

	// Bonus por ser una pregunta respondida
	if thought.IsQuestion() && thought.Score > 0.5 {
		value = 0.3
		rewardType = RewardAchievement
	}

	// Bonus por contenido social positivo
	if thought.IsSocial() && thought.Intensity > 0.4 {
		value = 0.2
		rewardType = RewardSocial
	}

	// Bonus por novedad
	if thought.HasRichAssociations() {
		value += 0.1
		rewardType = RewardNovelty
	}

	// Bonus por maestría (alta puntuación)
	if thought.Score > 0.7 {
		value += 0.15
		rewardType = RewardMastery
	}

	return RewardSignal{
		Type:      rewardType,
		Value:     clamp(value, 0, 1),
		Source:    thought.OriginalID,
		Timestamp: me.clock.NowMilli(),
	}
}

func (me *MotivationEngine) recordReward(reward RewardSignal) {
	me.mu.Lock()
	defer me.mu.Unlock()

	me.rewards = append(me.rewards, reward)
	if len(me.rewards) > 64 {
		me.rewards = me.rewards[1:]
	}

	// Actualizar estado emocional
	me.updateEmotionalState(reward.Value*0.1, reward.Value*0.05, reward.Value*0.02)

	// Actualizar struggling
	me.struggling *= 0.95 // Decaimiento natural

	// Actualizar propósito
	me.purpose.Intensity = clamp(me.purpose.Intensity+reward.Value*0.1, 0, 1)
}

func (me *MotivationEngine) updateStruggling(thought bus.ThoughtState) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Incrementar struggling por esfuerzo
	if thought.Intensity > 0.6 {
		me.struggling = clamp(me.struggling+0.02, 0, 1)
	}

	// Reducir struggling por éxito
	if thought.Score > 0.7 {
		me.struggling *= 0.9
	}
}

func (me *MotivationEngine) updatePurpose(thought bus.ThoughtState) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// El propósito se fortalece con intención social
	if thought.IsSocial() && thought.Intensity > 0.5 {
		me.purpose.Direction = clamp(me.purpose.Direction+0.02, 0, 1)
	}

	// El propósito se debilita con saturación
	state := me.stateReg.GetState()
	if state.Saturacion > 0.7 {
		me.purpose.Intensity *= 0.98
	}
}

func (me *MotivationEngine) updateEmotionalState(valenceDelta, arousalDelta, dominanceDelta float64) {
	me.emotionalState.Valence = clamp(me.emotionalState.Valence+valenceDelta, -1, 1)
	me.emotionalState.Arousal = clamp(me.emotionalState.Arousal+arousalDelta, 0, 1)
	me.emotionalState.Dominance = clamp(me.emotionalState.Dominance+dominanceDelta, 0, 1)
	me.emotionalState.LastUpdate = me.clock.NowMilli()

	// Sincronizar con state register
	state := me.stateReg.GetState()
	newValence := me.emotionalState.Valence*0.3 + state.Valencia*0.7

	me.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("emo_%d", me.clock.NowMilli()), Type: bus.Meta,
		Source: "motivation_engine", Target: "state_register",
		Priority: 50, Timestamp: me.clock.NowMilli(),
		Payload: []byte(fmt.Sprintf(`{"valencia":%.3f,"intensidad":%.3f}`, newValence, me.emotionalState.Arousal)),
		TTL: 1,
	})
}

func (me *MotivationEngine) createRewardFromMap(data map[string]any) RewardSignal {
	reward := RewardSignal{
		Timestamp: me.clock.NowMilli(),
	}

	if rewardType, ok := data["type"].(string); ok {
		switch rewardType {
		case "achievement":
			reward.Type = RewardAchievement
		case "social":
			reward.Type = RewardSocial
		case "purpose":
			reward.Type = RewardPurpose
		case "novelty":
			reward.Type = RewardNovelty
		case "mastery":
			reward.Type = RewardMastery
		}
	}

	if value, ok := data["value"].(float64); ok {
		reward.Value = clamp(value, 0, 1)
	}

	if source, ok := data["source"].(string); ok {
		reward.Source = source
	}

	return reward
}

func (me *MotivationEngine) updatePurposeFromMap(data map[string]any) {
	me.mu.Lock()
	defer me.mu.Unlock()

	if direction, ok := data["direction"].(float64); ok {
		me.purpose.Direction = clamp(direction, 0, 1)
	}
	if intensity, ok := data["intensity"].(float64); ok {
		me.purpose.Intensity = clamp(intensity, 0, 1)
	}
	if source, ok := data["source"].(string); ok {
		me.purpose.Source = source
	}
	me.purpose.LastSync = me.clock.NowMilli()
}

func (me *MotivationEngine) emitEmotionalState() {
	payload, _ := json.Marshal(map[string]any{
		"valence":   me.emotionalState.Valence,
		"arousal":   me.emotionalState.Arousal,
		"dominance": me.emotionalState.Dominance,
		"struggling": me.struggling,
		"purpose": map[string]any{
			"direction": me.purpose.Direction,
			"intensity": me.purpose.Intensity,
			"source":    me.purpose.Source,
		},
	})

	me.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("emo_state_%d", me.clock.NowMilli()), Type: bus.Meta,
		Source: "motivation_engine", Target: "meta_cognition",
		Priority: 50, Timestamp: me.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (me *MotivationEngine) GetStats() MotivationStats {
	me.mu.Lock()
	defer me.mu.Unlock()

	stats := me.stats
	stats.TotalRewards = len(me.rewards)
	stats.Struggling = me.struggling
	stats.PurposeStrength = me.purpose.Intensity
	stats.EmotionalBal = me.emotionalState.Valence

	if me.emotionalState.Arousal > 0 {
		stats.AvgArousal = me.emotionalState.Arousal
	}

	// Calcular tasa de recompensa
	if stats.TotalRewards > 0 {
		totalValue := 0.0
		for _, r := range me.rewards {
			totalValue += r.Value
		}
		stats.RewardRate = totalValue / float64(stats.TotalRewards)
	}

	return stats
}

func (me *MotivationEngine) GetEmotionalState() EmotionalState {
	me.mu.RLock()
	defer me.mu.RUnlock()
	return me.emotionalState
}

func (me *MotivationEngine) GetPurpose() PurposeVector {
	me.mu.RLock()
	defer me.mu.RUnlock()
	return *me.purpose
}

func (me *MotivationEngine) GetStruggling() float64 {
	me.mu.RLock()
	defer me.mu.RUnlock()
	return me.struggling
}

// GetRewardCurve retorna la curva de recompensa actual
func (me *MotivationEngine) GetRewardCurve() []float64 {
	me.mu.RLock()
	defer me.mu.RUnlock()

	curve := make([]float64, len(me.rewards))
	for i, r := range me.rewards {
		curve[i] = r.Value
	}
	return curve
}

// CalculateMotivationScore calcula el score de motivación actual
func (me *MotivationEngine) CalculateMotivationScore() float64 {
	me.mu.RLock()
	defer me.mu.RUnlock()

	// Componente de propósito
	purposeScore := me.purpose.Direction * me.purpose.Intensity * 0.4

	// Componente de recompensa
	rewardScore := 0.0
	if len(me.rewards) > 0 {
		for _, r := range me.rewards {
			rewardScore += r.Value
		}
		rewardScore /= float64(len(me.rewards))
	}
	rewardScore *= 0.3

	// Componente de struggling (inverso)
	struggleScore := (1.0 - me.struggling) * 0.2

	// Componente emocional
	emotionalScore := (me.emotionalState.Valence + 1.0) / 2.0 * 0.1

	return clamp(purposeScore+rewardScore+struggleScore+emotionalScore, 0, 1)
}
