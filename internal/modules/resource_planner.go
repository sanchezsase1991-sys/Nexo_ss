package modules

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// ResourcePlanner — SECCIÓN 7: Estimación y Asignación de Recursos
// Responsable de:
// 1. Estimación de tiempo y energía para tareas
// 2. Optimización de memoria de trabajo
// 3. Balance de recursos del sistema
// 4. Reservas para manejo de excepciones
// 5. Predicción de carga cognitiva

type ResourcePlanner struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	wm          *WorkingMemoryManager
	mu          sync.RWMutex
	resources   SystemResources
	estimates   []TaskEstimate
	maxEstimates int
	stats       ResourceStats
}

type SystemResources struct {
	CPULoad       float64 // 0-1
	MemoryUsage   float64 // 0-1
	Bandwidth     float64 // 0-1
	EnergyReserve float64 // 0-1
	LastUpdate    int64
}

type TaskEstimate struct {
	TaskID        string
	TimeEstimate  float64 // milisegundos
	EnergyCost    float64 // 0-1
	MemoryNeeded  float64 // 0-1
	Complexity    float64 // 0-1
	Priority      int
	Timestamp     int64
}

type ResourceStats struct {
	AvgCPULoad     float64
	AvgMemoryUsage float64
	TotalEstimates int
	AccuracyRate   float64
	Overestimate   float64
	Underestimate  float64
}

func NewResourcePlanner(stateReg *StateRegister, clock scheduler.Clock, wm *WorkingMemoryManager) *ResourcePlanner {
	return &ResourcePlanner{
		stateReg:     stateReg,
		clock:        clock,
		wm:           wm,
		resources:    SystemResources{EnergyReserve: 1.0},
		estimates:    make([]TaskEstimate, 0, 32),
		maxEstimates: 32,
	}
}

func (rp *ResourcePlanner) SetScheduler(s *scheduler.Scheduler) { rp.sched = s }

func (rp *ResourcePlanner) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Thought:
		rp.processThoughtForResource(pkt)
	case bus.Action:
		rp.processActionForResource(pkt)
	case bus.Meta:
		rp.processMeta(pkt)
	}
}

func (rp *ResourcePlanner) processThoughtForResource(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil {
		return
	}

	// Estimar recursos para el pensamiento
	estimate := rp.estimateTaskResources(thought)
	rp.recordEstimate(estimate)

	// Evaluar si hay suficientes recursos
	if !rp.hasSufficientResources(estimate) {
		// Emitir señal de advertencia
		rp.emitResourceWarning(estimate, pkt)
	}
}

func (rp *ResourcePlanner) processActionForResource(pkt bus.CognitivePacket) {
	var action map[string]any
	json.Unmarshal(pkt.Payload, &action)

	// Estimar recursos para la acción
	complexity := rp.estimateActionComplexity(action)
	estimate := TaskEstimate{
		TaskID:       fmt.Sprintf("action_%s", pkt.ID),
		EnergyCost:   complexity * 0.3,
		MemoryNeeded: complexity * 0.2,
		Complexity:   complexity,
		Priority:     pkt.Priority,
		Timestamp:    rp.clock.NowMilli(),
	}

	rp.recordEstimate(estimate)

	// Actualizar recursos del sistema
	rp.updateSystemResources(estimate)
}

func (rp *ResourcePlanner) processMeta(pkt bus.CognitivePacket) {
	var meta map[string]any
	json.Unmarshal(pkt.Payload, &meta)

	// Estimar tarea
	if estimateReq, ok := meta["estimate_task"].(map[string]any); ok {
		estimate := rp.createEstimateFromMap(estimateReq)
		rp.recordEstimate(estimate)

		payload, _ := json.Marshal(estimate)
		rp.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("est_%d", rp.clock.NowMilli()), Type: bus.Meta,
			Source: "resource_planner", Target: "control_planner",
			Priority: 50, Timestamp: rp.clock.NowMilli(),
			Payload: payload, TTL: 2,
		})
	}

	// Obtener recursos disponibles
	if getResources, ok := meta["get_resources"].(bool); ok && getResources {
		rp.emitResourceStatus()
	}

	// Optimizar memoria
	if optimize, ok := meta["optimize_memory"].(bool); ok && optimize {
		rp.optimizeMemory()
	}

	// Estadísticas
	if statsReq, ok := meta["request_stats"].(bool); ok && statsReq {
		stats := rp.GetStats()
		payload, _ := json.Marshal(stats)
		rp.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("rp_stats_%d", rp.clock.NowMilli()), Type: bus.Meta,
			Source: "resource_planner", Target: "meta_cognition",
			Priority: 40, Timestamp: rp.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

func (rp *ResourcePlanner) estimateTaskResources(thought bus.ThoughtState) TaskEstimate {
	// Estimación basada en complejidad del pensamiento
	complexity := thought.Intensity*0.4 + thought.Score*0.3
	if thought.HasRichAssociations() {
		complexity += 0.15
	}
	if thought.IsSocial() {
		complexity += 0.1
	}

	// Tiempo estimado (ms) - sobreestimado intencionalmente
	timeEstimate := complexity * 500.0

	// Energía estimada
	energyCost := complexity * 0.4

	// Memoria estimada
	memoryNeeded := complexity * 0.3

	return TaskEstimate{
		TaskID:       thought.OriginalID,
		TimeEstimate: timeEstimate,
		EnergyCost:   clamp(energyCost, 0, 1),
		MemoryNeeded: clamp(memoryNeeded, 0, 1),
		Complexity:   clamp(complexity, 0, 1),
		Priority:     50,
		Timestamp:    rp.clock.NowMilli(),
	}
}

func (rp *ResourcePlanner) estimateActionComplexity(action map[string]any) float64 {
	complexity := 0.3 // Base

	if _, ok := action["tool_name"]; ok {
		complexity += 0.2
	}
	if _, ok := action["social_response"]; ok {
		complexity += 0.15
	}
	if _, ok := action["memory_key"]; ok {
		complexity += 0.1
	}

	return clamp(complexity, 0, 1)
}

func (rp *ResourcePlanner) recordEstimate(estimate TaskEstimate) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.estimates = append(rp.estimates, estimate)
	if len(rp.estimates) > rp.maxEstimates {
		rp.estimates = rp.estimates[1:]
	}
}

func (rp *ResourcePlanner) hasSufficientResources(estimate TaskEstimate) bool {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	// Verificar energía
	if rp.resources.EnergyReserve < estimate.EnergyCost {
		return false
	}

	// Verificar memoria
	if rp.resources.MemoryUsage+estimate.MemoryNeeded > 0.9 {
		return false
	}

	// Verificar CPU
	if rp.resources.CPULoad > 0.8 {
		return false
	}

	return true
}

func (rp *ResourcePlanner) updateSystemResources(estimate TaskEstimate) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	// Actualizar uso de memoria
	rp.resources.MemoryUsage = clamp(rp.resources.MemoryUsage+estimate.MemoryNeeded*0.1, 0, 1)

	// Actualizar reserva de energía
	rp.resources.EnergyReserve = clamp(rp.resources.EnergyReserve-estimate.EnergyCost*0.05, 0, 1)

	// Actualizar CPU (basado en complejidad)
	rp.resources.CPULoad = clamp(estimate.Complexity*0.3+rp.resources.CPULoad*0.7, 0, 1)

	rp.resources.LastUpdate = rp.clock.NowMilli()
}

func (rp *ResourcePlanner) emitResourceWarning(estimate TaskEstimate, pkt bus.CognitivePacket) {
	warning := map[string]any{
		"warning":       "insufficient_resources",
		"task_id":       estimate.TaskID,
		"energy_needed": estimate.EnergyCost,
		"energy_avail":  rp.resources.EnergyReserve,
		"memory_needed": estimate.MemoryNeeded,
		"memory_avail":  1.0 - rp.resources.MemoryUsage,
	}

	payload, _ := json.Marshal(warning)
	rp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("res_warn_%s", pkt.ID), Type: bus.Meta,
		Source: "resource_planner", Target: "fatigue_compensator",
		Priority: 80, Timestamp: rp.clock.NowMilli(),
		Payload: payload, TTL: 3,
	})
}

func (rp *ResourcePlanner) emitResourceStatus() {
	rp.mu.RLock()
	status := rp.resources
	rp.mu.RUnlock()

	payload, _ := json.Marshal(map[string]any{
		"cpu_load":      status.CPULoad,
		"memory_usage":  status.MemoryUsage,
		"bandwidth":     status.Bandwidth,
		"energy_reserve": status.EnergyReserve,
	})

	rp.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("res_status_%d", rp.clock.NowMilli()), Type: bus.Meta,
		Source: "resource_planner", Target: "meta_cognition",
		Priority: 40, Timestamp: rp.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (rp *ResourcePlanner) optimizeMemory() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	// Reducir uso de memoria si está alto
	if rp.resources.MemoryUsage > 0.7 {
		rp.resources.MemoryUsage *= 0.9
	}

	// Recuperar energía si hay exceso
	if rp.resources.EnergyReserve > 0.8 {
		rp.resources.EnergyReserve = clamp(rp.resources.EnergyReserve-0.05, 0, 1)
	}
}

func (rp *ResourcePlanner) createEstimateFromMap(data map[string]any) TaskEstimate {
	estimate := TaskEstimate{
		Timestamp: rp.clock.NowMilli(),
	}

	if taskID, ok := data["task_id"].(string); ok {
		estimate.TaskID = taskID
	}
	if timeEst, ok := data["time_estimate"].(float64); ok {
		estimate.TimeEstimate = timeEst
	}
	if energy, ok := data["energy_cost"].(float64); ok {
		estimate.EnergyCost = clamp(energy, 0, 1)
	}
	if memory, ok := data["memory_needed"].(float64); ok {
		estimate.MemoryNeeded = clamp(memory, 0, 1)
	}
	if complexity, ok := data["complexity"].(float64); ok {
		estimate.Complexity = clamp(complexity, 0, 1)
	}
	if priority, ok := data["priority"].(float64); ok {
		estimate.Priority = int(priority)
	}

	return estimate
}

func (rp *ResourcePlanner) GetStats() ResourceStats {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	stats := rp.stats
	stats.TotalEstimates = len(rp.estimates)

	// Calcular promedios
	if stats.TotalEstimates > 0 {
		totalCPU := 0.0
		totalMemory := 0.0
		for _, est := range rp.estimates {
			totalCPU += est.Complexity
			totalMemory += est.MemoryNeeded
		}
		stats.AvgCPULoad = totalCPU / float64(stats.TotalEstimates)
		stats.AvgMemoryUsage = totalMemory / float64(stats.TotalEstimates)
	}

	return stats
}

func (rp *ResourcePlanner) GetResources() SystemResources {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.resources
}

func (rp *ResourcePlanner) GetEstimates() []TaskEstimate {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	estimates := make([]TaskEstimate, len(rp.estimates))
	copy(estimates, rp.estimates)
	return estimates
}

// CalculateCognitiveLoad calcula la carga cognitiva actual
func (rp *ResourcePlanner) CalculateCognitiveLoad() float64 {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	// Carga = promedio ponderado de todos los recursos
	load := rp.resources.CPULoad*0.3 +
		rp.resources.MemoryUsage*0.3 +
		rp.resources.Bandwidth*0.2 +
		(1.0-rp.resources.EnergyReserve)*0.2

	return clamp(load, 0, 1)
}

// PredictCognitiveCapacity predice la capacidad cognitiva futura
func (rp *ResourcePlanner) PredictCognitiveCapacity(timeHorizonMs int64) float64 {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	currentLoad := rp.CalculateCognitiveLoad()

	// Factor de recuperación con el tiempo
	recoveryFactor := 1.0 - math.Exp(-float64(timeHorizonMs)/10000.0)

	// Capacidad predicha
	predicted := (1.0-currentLoad)*recoveryFactor + 0.1

	return clamp(predicted, 0, 1)
}
