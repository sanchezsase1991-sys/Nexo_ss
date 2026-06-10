package modules

import (
	"fmt"
	"sync"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// ResourceEstimate representa una estimación de recursos
type ResourceEstimate struct {
	TaskID           string  `json:"task_id"`
	TimeObjective    float64 `json:"time_objective"`
	TimePreparation  float64 `json:"time_preparation"`
	TimePostProcess  float64 `json:"time_post_process"`
	CostEmotional    float64 `json:"cost_emotional"`
	CostSocial       float64 `json:"cost_social"`
	ReserveImprevist float64 `json:"reserve_imprevist"`
	TotalEstimated   float64 `json:"total_estimated"`
	Overestimation   bool    `json:"overestimation"`
}

// ResourceEstimator implementa la estimación de recursos del perfil
type ResourceEstimator struct {
	sched    *scheduler.Scheduler
	clock    scheduler.Clock
	stateReg *StateRegister
	mu       sync.Mutex
	history  []ResourceEstimate
}

func NewResourceEstimator(stateReg *StateRegister, clock scheduler.Clock) *ResourceEstimator {
	return &ResourceEstimator{
		stateReg: stateReg,
		clock:    clock,
		history:  make([]ResourceEstimate, 0, 100),
	}
}

func (re *ResourceEstimator) SetScheduler(s *scheduler.Scheduler) { re.sched = s }

// Estimate calcula la estimación de recursos para una tarea
func (re *ResourceEstimator) Estimate(taskID string, objectiveTime float64, hasPurpose bool, hasSocialInteraction bool) *ResourceEstimate {
	state := re.stateReg.GetState()

	estimate := &ResourceEstimate{
		TaskID:        taskID,
		TimeObjective: objectiveTime,
	}

	// Subestima tiempo objetivo
	estimate.TimeObjective *= 0.8

	// Sobreestima preparación mental
	estimate.TimePreparation = objectiveTime * 0.3
	if !hasPurpose { estimate.TimePreparation *= 1.5 }

	// Sobreestima post-procesamiento
	estimate.TimePostProcess = objectiveTime * 0.2

	// Muy sobreestima costo emocional
	estimate.CostEmotional = 0.3
	if !hasPurpose { estimate.CostEmotional = 0.6 }
	if state.Saturacion > 0.5 { estimate.CostEmotional += 0.2 }

	// Sobreestima costo social
	if hasSocialInteraction {
		estimate.CostSocial = 0.3
		if state.PresionSocial > 0.5 { estimate.CostSocial += 0.2 }
	}

	// Alta reserva para imprevistos
	estimate.ReserveImprevist = objectiveTime * 0.25

	// Calcular total
	estimate.TotalEstimated = estimate.TimeObjective +
		estimate.TimePreparation +
		estimate.TimePostProcess +
		estimate.CostEmotional*objectiveTime +
		estimate.CostSocial*objectiveTime +
		estimate.ReserveImprevist

	// Marcar sobreestimación
	estimate.Overestimation = estimate.TotalEstimated > objectiveTime*2

	// Guardar historial
	re.mu.Lock()
	re.history = append(re.history, *estimate)
	if len(re.history) > 100 { re.history = re.history[1:] }
	re.mu.Unlock()

	// Emitir estimación como Meta
	payload, _ := json.Marshal(estimate)
	re.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("est_%s", taskID),
		Type:      bus.Meta,
		Source:    "resource_estimator",
		Target:    "control_planner",
		Priority:  50,
		Timestamp: re.clock.NowMilli(),
		Payload:   payload,
		Tags:      []string{"resource_estimate"},
		TTL:       3,
	})

	return estimate
}

func (re *ResourceEstimator) Handle(pkt bus.CognitivePacket) {}

func (re *ResourceEstimator) GetHistory() []ResourceEstimate {
	re.mu.Lock()
	defer re.mu.Unlock()
	result := make([]ResourceEstimate, len(re.history))
	copy(result, re.history)
	return result
}
