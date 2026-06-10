package modules

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// GoalManager — SECCIÓN 5: Módulo de Metas y Propósito
// Responsable de:
// 1. Definición de metas y objetivos del sistema
// 2. Establecimiento de prioridades entre metas
// 3. Seguimiento de progreso hacia metas
// 4. Resolución de conflictos de metas
// 5. Alineación de metas con valores del sistema

type GoalManager struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	wm          *WorkingMemoryManager
	mu          sync.RWMutex
	goals       []Goal
	activeGoal  *Goal
	goalHistory []GoalResult
	maxGoals    int
	maxHistory  int
	stats       GoalStats
}

type Goal struct {
	ID           string
	Name         string
	Description  string
	Priority     int
	Progress     float64
	Status       GoalStatus
	CreatedAt    int64
	UpdatedAt    int64
	Deadline     int64
	Dependencies []string
	ParentGoal   string
	SubGoals     []string
	Tags         []string
	ValueAlign   float64 // Alineación con valores del sistema (0-1)
	SocialImpact float64 // Impacto social esperado (0-1)
}

type GoalStatus int

const (
	GoalPending GoalStatus = iota
	GoalActive
	GoalInProgress
	GoalCompleted
	GoalFailed
	GoalBlocked
	GoalDeferred
)

func (gs GoalStatus) String() string {
	switch gs {
	case GoalPending:
		return "PENDING"
	case GoalActive:
		return "ACTIVE"
	case GoalInProgress:
		return "IN_PROGRESS"
	case GoalCompleted:
		return "COMPLETED"
	case GoalFailed:
		return "FAILED"
	case GoalBlocked:
		return "BLOCKED"
	case GoalDeferred:
		return "DEFERRED"
	default:
		return "UNKNOWN"
	}
}

type GoalResult struct {
	GoalID    string
	Success   bool
	Duration  int64
	FinalProg float64
	Timestamp int64
}

type GoalStats struct {
	TotalGoals     int
	Active         int
	Completed      int
	Failed         int
	AvgProgress    float64
	AvgResolution  float64
	ConflictCount  int
}

type GoalConflict struct {
	GoalA     string
	GoalB     string
	Type      ConflictType
	Severity  float64
	Resolved  bool
}

type ConflictType int

const (
	ConflictResource ConflictType = iota
	ConflictTemporal
	ConflictPriority
	ConflictValue
)

func (ct ConflictType) String() string {
	switch ct {
	case ConflictResource:
		return "RESOURCE"
	case ConflictTemporal:
		return "TEMPORAL"
	case ConflictPriority:
		return "PRIORITY"
	case ConflictValue:
		return "VALUE"
	default:
		return "UNKNOWN"
	}
}

func NewGoalManager(stateReg *StateRegister, clock scheduler.Clock, wm *WorkingMemoryManager) *GoalManager {
	return &GoalManager{
		stateReg:    stateReg,
		clock:       clock,
		wm:          wm,
		goals:       make([]Goal, 0, 32),
		goalHistory: make([]GoalResult, 0, 128),
		maxGoals:    32,
		maxHistory:  128,
	}
}

func (gm *GoalManager) SetScheduler(s *scheduler.Scheduler) { gm.sched = s }

func (gm *GoalManager) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Thought:
		gm.processThoughtForGoal(pkt)
	case bus.Action:
		gm.processActionResult(pkt)
	case bus.Meta:
		gm.processMeta(pkt)
	case bus.Memory:
		gm.processMemoryUpdate(pkt)
	}
}

func (gm *GoalManager) processThoughtForGoal(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil {
		return
	}

	// Evaluar si el pensamiento contiene una meta o está relacionado con una meta
	goalRelevance := gm.evaluateGoalRelevance(thought)

	if goalRelevance > 0.5 {
		// Crear o actualizar meta
		goal := gm.createGoalFromThought(thought, goalRelevance)
		gm.addGoal(goal)
	}
}

func (gm *GoalManager) processActionResult(pkt bus.CognitivePacket) {
	var result map[string]any
	json.Unmarshal(pkt.Payload, &result)

	// Verificar si esta acción estaba asociada a una meta
	if goalID, ok := result["goal_id"].(string); ok {
		gm.updateGoalProgress(goalID, result)
	}
}

func (gm *GoalManager) processMeta(pkt bus.CognitivePacket) {
	var meta map[string]any
	json.Unmarshal(pkt.Payload, &meta)

	// Crear meta
	if createReq, ok := meta["create_goal"].(map[string]any); ok {
		goal := gm.createGoalFromMap(createReq)
		gm.addGoal(goal)
	}

	// Actualizar progreso de meta
	if updateReq, ok := meta["update_goal"].(map[string]any); ok {
		if goalID, ok := updateReq["id"].(string); ok {
			gm.updateGoalProgress(goalID, updateReq)
		}
	}

	// Completar meta
	if completeReq, ok := meta["complete_goal"].(map[string]any); ok {
		if goalID, ok := completeReq["id"].(string); ok {
			gm.completeGoal(goalID)
		}
	}

	// Obtener meta activa
	if getActive, ok := meta["get_active"].(bool); ok && getActive {
		gm.emitActiveGoal()
	}

	// Estadísticas
	if statsReq, ok := meta["request_stats"].(bool); ok && statsReq {
		stats := gm.GetStats()
		payload, _ := json.Marshal(stats)
		gm.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("goal_stats_%d", gm.clock.NowMilli()), Type: bus.Meta,
			Source: "goal_manager", Target: "meta_cognition",
			Priority: 40, Timestamp: gm.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

func (gm *GoalManager) processMemoryUpdate(pkt bus.CognitivePacket) {
	// Actualizar progreso de metas basado en eventos de memoria
	var memEvent map[string]any
	json.Unmarshal(pkt.Payload, &memEvent)

	if eventType, ok := memEvent["event"].(string); ok {
		switch eventType {
		case "eviction":
			gm.handleMemoryEviction()
			case "consolidation":
			gm.handleMemoryConsolidation()
		}
	}
}

func (gm *GoalManager) evaluateGoalRelevance(thought bus.ThoughtState) float64 {
	score := 0.0

	// Bonus por ser una pregunta (posible objetivo)
	if thought.IsQuestion() {
		score += 0.3
	}

	// Bonus por urgencia (posible deadline)
	if thought.IsUrgent() {
		score += 0.2
	}

	// Bonus por relevancia social (impacto en otros)
	if thought.IsSocial() {
		score += 0.15
	}

	// Bonus por intensidad alta
	score += thought.Intensity * 0.2

	// Bonus por tener asociaciones ricas
	if thought.HasRichAssociations() {
		score += 0.1
	}

	// Penalización por saturación
	state := gm.stateReg.GetState()
	score *= (1.0 - state.Saturacion*0.2)

	return clamp(score, 0, 1)
}

func (gm *GoalManager) createGoalFromThought(thought bus.ThoughtState, relevance float64) Goal {
	return Goal{
		ID:           fmt.Sprintf("goal_%s", thought.OriginalID),
		Name:         truncateStr(thought.Payload, 50),
		Description:  thought.Payload,
		Priority:     int(relevance * 100),
		Progress:     0,
		Status:       GoalPending,
		CreatedAt:    gm.clock.NowMilli(),
		UpdatedAt:    gm.clock.NowMilli(),
		Dependencies: make([]string, 0),
		SubGoals:     make([]string, 0),
		Tags:         thought.Tags,
		ValueAlign:   relevance * 0.8,
		SocialImpact: relevance * 0.6,
	}
}

func (gm *GoalManager) createGoalFromMap(data map[string]any) Goal {
	goal := Goal{
		ID:           fmt.Sprintf("goal_%d", gm.clock.NowMilli()),
		Status:       GoalPending,
		CreatedAt:    gm.clock.NowMilli(),
		UpdatedAt:    gm.clock.NowMilli(),
		Dependencies: make([]string, 0),
		SubGoals:     make([]string, 0),
	}

	if name, ok := data["name"].(string); ok {
		goal.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		goal.Description = desc
	}
	if priority, ok := data["priority"].(float64); ok {
		goal.Priority = int(priority)
	}
	if deadline, ok := data["deadline"].(float64); ok {
		goal.Deadline = int64(deadline)
	}
	if deps, ok := data["dependencies"].([]string); ok {
		goal.Dependencies = deps
	}
	if tags, ok := data["tags"].([]string); ok {
		goal.Tags = tags
	}
	if va, ok := data["value_alignment"].(float64); ok {
		goal.ValueAlign = va
	}
	if si, ok := data["social_impact"].(float64); ok {
		goal.SocialImpact = si
	}

	return goal
}

func (gm *GoalManager) addGoal(goal Goal) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Verificar capacidad
	if len(gm.goals) >= gm.maxGoals {
		// Remover la meta de menor prioridad
		lowestIdx := 0
		lowestPri := gm.goals[0].Priority
		for i, g := range gm.goals {
			if g.Priority < lowestPri {
				lowestPri = g.Priority
				lowestIdx = i
			}
		}
		gm.goals = append(gm.goals[:lowestIdx], gm.goals[lowestIdx+1:]...)
	}

	// Agregar meta
	gm.goals = append(gm.goals, goal)

	// Actualizar meta activa si es necesario
	if gm.activeGoal == nil || goal.Priority > gm.activeGoal.Priority {
		gm.activeGoal = &gm.goals[len(gm.goals)-1]
		gm.activeGoal.Status = GoalActive
	}
}

func (gm *GoalManager) updateGoalProgress(goalID string, update map[string]any) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	for i := range gm.goals {
		if gm.goals[i].ID == goalID {
			if progress, ok := update["progress"].(float64); ok {
				gm.goals[i].Progress = clamp(progress, 0, 1)
			}
			if status, ok := update["status"].(string); ok {
				switch status {
				case "in_progress":
					gm.goals[i].Status = GoalInProgress
				case "completed":
					gm.goals[i].Status = GoalCompleted
				case "failed":
					gm.goals[i].Status = GoalFailed
				case "blocked":
					gm.goals[i].Status = GoalBlocked
				}
			}
			gm.goals[i].UpdatedAt = gm.clock.NowMilli()
			break
		}
	}
}

func (gm *GoalManager) completeGoal(goalID string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	for i := range gm.goals {
		if gm.goals[i].ID == goalID {
			gm.goals[i].Status = GoalCompleted
			gm.goals[i].Progress = 1.0
			gm.goals[i].UpdatedAt = gm.clock.NowMilli()

			// Registrar en historial
			result := GoalResult{
				GoalID:    goalID,
				Success:   true,
				Duration:  gm.goals[i].UpdatedAt - gm.goals[i].CreatedAt,
				FinalProg: 1.0,
				Timestamp: gm.clock.NowMilli(),
			}
			gm.goalHistory = append(gm.goalHistory, result)
			if len(gm.goalHistory) > gm.maxHistory {
				gm.goalHistory = gm.goalHistory[1:]
			}

			// Actualizar estadísticas
			gm.stats.Completed++
			gm.stats.Active--

			// Si esta era la meta activa, buscar la siguiente
			if gm.activeGoal != nil && gm.activeGoal.ID == goalID {
				gm.activeGoal = nil
				gm.findNextActiveGoal()
			}
			break
		}
	}
}

func (gm *GoalManager) findNextActiveGoal() {
	var bestGoal *Goal
	bestPriority := -1

	for i := range gm.goals {
		if gm.goals[i].Status == GoalPending || gm.goals[i].Status == GoalInProgress {
			if gm.goals[i].Priority > bestPriority {
				bestPriority = gm.goals[i].Priority
				bestGoal = &gm.goals[i]
			}
		}
	}

	if bestGoal != nil {
		bestGoal.Status = GoalActive
		gm.activeGoal = bestGoal
	}
}

func (gm *GoalManager) handleMemoryEviction() {
	// Cuando hay evicción de memoria, verificar si afecta metas activas
	gm.mu.Lock()
	defer gm.mu.Unlock()

	for i := range gm.goals {
		if gm.goals[i].Status == GoalInProgress {
			// Reducir progreso si se pierde contexto
			gm.goals[i].Progress *= 0.95
		}
	}
}

func (gm *GoalManager) handleMemoryConsolidation() {
	// Cuando se consolida memoria, verificar progreso de metas
	gm.mu.Lock()
	defer gm.mu.Unlock()

	for i := range gm.goals {
		if gm.goals[i].Status == GoalInProgress {
			// Incrementar progreso si se consolida conocimiento
			gm.goals[i].Progress = clamp(gm.goals[i].Progress+0.02, 0, 1)
		}
	}
}

func (gm *GoalManager) emitActiveGoal() {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if gm.activeGoal == nil {
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"id":          gm.activeGoal.ID,
		"name":        gm.activeGoal.Name,
		"progress":    gm.activeGoal.Progress,
		"status":      gm.activeGoal.Status.String(),
		"priority":    gm.activeGoal.Priority,
		"value_align": gm.activeGoal.ValueAlign,
	})

	gm.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("active_goal_%d", gm.clock.NowMilli()), Type: bus.Meta,
		Source: "goal_manager", Target: "control_planner",
		Priority: 60, Timestamp: gm.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (gm *GoalManager) GetStats() GoalStats {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	stats := gm.stats
	stats.TotalGoals = len(gm.goals)
	stats.Active = 0
	stats.Completed = 0
	stats.Failed = 0

	totalProgress := 0.0
	for _, goal := range gm.goals {
		switch goal.Status {
		case GoalActive, GoalInProgress:
			stats.Active++
		case GoalCompleted:
			stats.Completed++
		case GoalFailed:
			stats.Failed++
		}
		totalProgress += goal.Progress
	}

	if stats.TotalGoals > 0 {
		stats.AvgProgress = totalProgress / float64(stats.TotalGoals)
	}

	return stats
}

func (gm *GoalManager) GetActiveGoal() *Goal {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.activeGoal
}

func (gm *GoalManager) GetGoalByID(id string) *Goal {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	for i := range gm.goals {
		if gm.goals[i].ID == id {
			return &gm.goals[i]
		}
	}
	return nil
}

func (gm *GoalManager) GetAllGoals() []Goal {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	goals := make([]Goal, len(gm.goals))
	copy(goals, gm.goals)
	return goals
}
