package modules

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// MentalModel representa un modelo mental del escenario
type MentalModel struct {
	ID          string                 `json:"id"`
	Scenario    string                 `json:"scenario"`
	Confidence  float64                `json:"confidence"`
	LastUpdated int64                  `json:"last_updated"`
	Revision    int                    `json:"revision"`
	Data        map[string]interface{} `json:"data"`
}

// RepresentationUpdater actualiza representaciones en tiempo real
type RepresentationUpdater struct {
	sched      *scheduler.Scheduler
	clock      scheduler.Clock
	stateReg   *StateRegister
	models     map[string]*MentalModel
	mu         sync.RWMutex
	updateCount int
}

func NewRepresentationUpdater(stateReg *StateRegister, clock scheduler.Clock) *RepresentationUpdater {
	return &RepresentationUpdater{
		stateReg: stateReg,
		clock:    clock,
		models:   make(map[string]*MentalModel),
	}
}

func (ru *RepresentationUpdater) SetScheduler(s *scheduler.Scheduler) { ru.sched = s }

// Handle procesa pensamientos para actualizar modelos mentales
func (ru *RepresentationUpdater) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }

	thought, err := pkt.AsThoughtState()
	if err != nil { return }

	// Construir o actualizar modelo mental del escenario
	ru.updateModel(thought)

	// Verificar inconsistencias
	if ru.detectInconsistency(thought) {
		ru.restructureModel(thought)
	}
}

func (ru *RepresentationUpdater) updateModel(thought bus.ThoughtState) {
	ru.mu.Lock()
	defer ru.mu.Unlock()

	modelID := fmt.Sprintf("model_%s", thought.Source)
	model, exists := ru.models[modelID]

	if !exists {
		model = &MentalModel{
			ID:       modelID,
			Scenario: thought.Payload,
			Data:     make(map[string]interface{}),
		}
		ru.models[modelID] = model
	}

	model.Confidence = 0.7
	model.LastUpdated = ru.clock.NowMilli()
	model.Revision++
	model.Data["last_payload"] = thought.Payload
	model.Data["intensity"] = thought.Intensity
	model.Data["tags"] = thought.Tags

	ru.updateCount++
}

func (ru *RepresentationUpdater) detectInconsistency(thought bus.ThoughtState) bool {
	ru.mu.RLock()
	defer ru.mu.RUnlock()

	modelID := fmt.Sprintf("model_%s", thought.Source)
	model, exists := ru.models[modelID]
	if !exists { return false }

	// Detectar cambio en intensidad que contradice el modelo
	if lastIntensity, ok := model.Data["intensity"].(float64); ok {
		if abs(thought.Intensity-lastIntensity) > 0.3 {
			return true
		}
	}

	return false
}

func (ru *RepresentationUpdater) restructureModel(thought bus.ThoughtState) {
	ru.mu.Lock()
	defer ru.mu.Unlock()

	modelID := fmt.Sprintf("model_%s", thought.Source)

	// Reestructurar: descartar interpretación anterior y construir nueva
	newModel := &MentalModel{
		ID:          modelID,
		Scenario:    fmt.Sprintf("[Reestructurado v%d] %s", ru.updateCount, thought.Payload),
		Confidence:  0.5, // Confianza reducida tras reestructuración
		LastUpdated: ru.clock.NowMilli(),
		Revision:    ru.updateCount,
		Data: map[string]interface{}{
			"restructured_from": thought.Payload,
			"restructure_reason": "inconsistency_detected",
			"intensity":         thought.Intensity,
		},
	}
	ru.models[modelID] = newModel

	// Emitir evento de reestructuración
	payload, _ := json.Marshal(newModel)
	ru.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("restruct_%s", thought.OriginalID),
		Type:      bus.Meta,
		Source:    "representation_updater",
		Target:    "working_memory",
		Priority:  60,
		Timestamp: ru.clock.NowMilli(),
		Payload:   payload,
		Tags:      []string{"mental_model", "restructured"},
		TTL:       3,
	})

	log.Printf("[RepresentationUpdater] Model %s restructured (revision %d)", modelID, ru.updateCount)
}

func (ru *RepresentationUpdater) GetModel(source string) *MentalModel {
	ru.mu.RLock()
	defer ru.mu.RUnlock()
	modelID := fmt.Sprintf("model_%s", source)
	return ru.models[modelID]
}

func abs(x float64) float64 {
	if x < 0 { return -x }
	return x
}
