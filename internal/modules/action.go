package modules

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// ActionExecutor — SECCIÓN 4: Módulo de Ejecución y Planificación Motora
// Responsable de:
// 1. Ejecución de movimientos voluntarios (acciones del sistema)
// 2. Selección entre alternativas motoras
// 3. Coordinación de músculos orofaciales para el habla
// 4. Buffer de salida con retardo voluntario
// 5. Monitoreo de ejecución en tiempo real

type ActionExecutor struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	wm          *WorkingMemoryManager
	perception  *PerceptionGate
	mu          sync.Mutex
	actionQueue []ActionItem
	execHistory []ActionResult
	maxQueue    int
	maxHistory  int
	stats       ActionStats
}

type ActionItem struct {
	ID          string
	Type        ActionType
	Priority    int
	Payload     map[string]any
	Prepared    bool
	CreatedAt   int64
	ExecuteAt   int64
	RetryCount  int
	MaxRetries  int
}

type ActionItemState int

const (
	ActionPending ActionItemState = iota
	ActionPreparing
	ActionExecuting
	ActionCompleted
	ActionFailed
	ActionAborted
)

type ActionResult struct {
	ActionID    string
	Type        ActionType
	Success     bool
	Latency     int64
	Error       string
	Timestamp   int64
}

type ActionStats struct {
	TotalActions    int
	Executed        int
	Failed          int
	AvgLatency      float64
	QueueDepth      int
	RetryRate       float64
}

type ActionType int

const (
	ActionOutput ActionType = iota
	ActionToolExecution
	ActionMemoryStore
	ActionStateUpdate
	ActionSocialResponse
	ActionInhibition
)

func (at ActionType) String() string {
	switch at {
	case ActionOutput:
		return "OUTPUT"
	case ActionToolExecution:
		return "TOOL"
	case ActionMemoryStore:
		return "MEMORY"
	case ActionStateUpdate:
		return "STATE"
	case ActionSocialResponse:
		return "SOCIAL"
	case ActionInhibition:
		return "INHIBIT"
	default:
		return "UNKNOWN"
	}
}

func NewActionExecutor(stateReg *StateRegister, clock scheduler.Clock, wm *WorkingMemoryManager, perception *PerceptionGate) *ActionExecutor {
	return &ActionExecutor{
		stateReg:   stateReg,
		clock:      clock,
		wm:         wm,
		perception: perception,
		actionQueue: make([]ActionItem, 0, 64),
		execHistory: make([]ActionResult, 0, 256),
		maxQueue:    64,
		maxHistory:  256,
	}
}

func (ae *ActionExecutor) SetScheduler(s *scheduler.Scheduler) { ae.sched = s }

func (ae *ActionExecutor) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Action:
		ae.processActionRequest(pkt)
	case bus.Thought:
		ae.processThoughtForAction(pkt)
	case bus.Meta:
		ae.processMeta(pkt)
	case bus.ToolResult:
		ae.processToolResult(pkt)
	}
}

func (ae *ActionExecutor) processActionRequest(pkt bus.CognitivePacket) {
	var request map[string]any
	json.Unmarshal(pkt.Payload, &request)

	// Crear item de acción
	actionType := ae.determineActionType(request)
	item := ActionItem{
		ID:         fmt.Sprintf("act_%s", pkt.ID),
		Type:       actionType,
		Priority:   pkt.Priority,
		Payload:    request,
		Prepared:   false,
		CreatedAt:  ae.clock.NowMilli(),
		ExecuteAt:  0,
		RetryCount: 0,
		MaxRetries: 3,
	}

	// Evaluar si se debe ejecutar inmediatamente
	state := ae.stateReg.GetState()
	if ae.shouldExecuteNow(item, state) {
		ae.executeAction(item, pkt)
	} else {
		ae.enqueueAction(item)
	}
}

func (ae *ActionExecutor) processThoughtForAction(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil {
		return
	}

	// Si el pensamiento tiene alta intensidad, crear item de acción
	if thought.Intensity > 0.6 {
		item := ActionItem{
			ID:         fmt.Sprintf("act_th_%s", thought.OriginalID),
			Type:       ActionOutput,
			Priority:   80,
			Payload:    map[string]any{"thought_id": thought.OriginalID, "content": thought.Payload},
			Prepared:   false,
			CreatedAt:  ae.clock.NowMilli(),
			RetryCount: 0,
			MaxRetries: 2,
		}
		ae.enqueueAction(item)
	}
}

func (ae *ActionExecutor) processMeta(pkt bus.CognitivePacket) {
	var meta map[string]any
	json.Unmarshal(pkt.Payload, &meta)

	// Flush de cola de acciones
	if flush, ok := meta["flush_actions"].(bool); ok && flush {
		ae.flushActionQueue()
	}

	// Estadísticas
	if statsReq, ok := meta["request_stats"].(bool); ok && statsReq {
		stats := ae.GetStats()
		payload, _ := json.Marshal(stats)
		ae.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("act_stats_%d", ae.clock.NowMilli()), Type: bus.Meta,
			Source: "action_executor", Target: "meta_cognition",
			Priority: 40, Timestamp: ae.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

func (ae *ActionExecutor) processToolResult(pkt bus.CognitivePacket) {
	var result map[string]any
	json.Unmarshal(pkt.Payload, &result)

	// Registrar resultado en historial
	actionResult := ActionResult{
		ActionID:  fmt.Sprintf("tool_%s", pkt.ID),
		Type:      ActionToolExecution,
		Success:   result["success"].(bool),
		Latency:   result["latency"].(int64),
		Timestamp: ae.clock.NowMilli(),
	}

	ae.mu.Lock()
	ae.execHistory = append(ae.execHistory, actionResult)
	if len(ae.execHistory) > ae.maxHistory {
		ae.execHistory = ae.execHistory[1:]
	}
	ae.mu.Unlock()

	// Actualizar estadísticas
	ae.updateStats(actionResult)
}

func (ae *ActionExecutor) shouldExecuteNow(item ActionItem, state SystemState) bool {
	// Siempre ejecutar inhibiciones
	if item.Type == ActionInhibition {
		return true
	}

	// Si el sistema está saturado, no ejecutar nada nuevo
	if state.Saturacion > 0.8 {
		return false
	}

	// Si la prioridad es alta y hay capacidad
	if item.Priority > 80 && state.CognitiveCapacity() > 0.5 {
		return true
	}

	// Si es una acción social con presión social
	if item.Type == ActionSocialResponse && state.PresionSocial > 0.5 {
		return true
	}

	return false
}

func (ae *ActionExecutor) determineActionType(request map[string]any) ActionType {
	if _, ok := request["tool_name"]; ok {
		return ActionToolExecution
	}
	if _, ok := request["memory_key"]; ok {
		return ActionMemoryStore
	}
	if _, ok := request["state_update"]; ok {
		return ActionStateUpdate
	}
	if _, ok := request["inhibit"]; ok {
		return ActionInhibition
	}
	if _, ok := request["social_response"]; ok {
		return ActionSocialResponse
	}
	return ActionOutput
}

func (ae *ActionExecutor) executeAction(item ActionItem, pkt bus.CognitivePacket) {
	start := ae.clock.NowMilli()

	// Preparar la acción
	prepared := ae.prepareAction(item)
	if !prepared {
		ae.recordFailure(item, "preparation_failed")
		return
	}

	// Ejecutar la acción
	success := ae.executePreparedAction(item)

	// Registrar resultado
	latency := ae.clock.NowMilli() - start
	ae.recordResult(item, success, latency)

	// Emitir resultado
	ae.emitActionResult(item, success, pkt)
}

func (ae *ActionExecutor) prepareAction(item ActionItem) bool {
	// Verificar que hay recursos suficientes
	state := ae.stateReg.GetState()
	if state.Saturacion > 0.85 {
		return false
	}

	// Preparar según tipo
	switch item.Type {
	case ActionOutput:
		return ae.prepareOutputAction(item)
	case ActionToolExecution:
		return ae.prepareToolAction(item)
	case ActionMemoryStore:
		return ae.prepareMemoryAction(item)
	case ActionStateUpdate:
		return ae.prepareStateAction(item)
	case ActionSocialResponse:
		return ae.prepareSocialAction(item)
	case ActionInhibition:
		return ae.prepareInhibitionAction(item)
	}

	return false
}

func (ae *ActionExecutor) prepareOutputAction(item ActionItem) bool {
	// Verificar que el output está calibrado
	if _, ok := item.Payload["calibrated"]; !ok {
		item.Payload["calibrated"] = false
	}
	return true
}

func (ae *ActionExecutor) prepareToolAction(item ActionItem) bool {
	// Verificar que la herramienta está disponible
	if _, ok := item.Payload["tool_name"]; !ok {
		return false
	}
	return true
}

func (ae *ActionExecutor) prepareMemoryAction(item ActionItem) bool {
	// Verificar que hay datos para almacenar
	if _, ok := item.Payload["memory_key"]; !ok {
		return false
	}
	return true
}

func (ae *ActionExecutor) prepareStateAction(item ActionItem) bool {
	return true
}

func (ae *ActionExecutor) prepareSocialAction(item ActionItem) bool {
	// Verificar que hay contexto social
	if _, ok := item.Payload["social_context"]; !ok {
		return false
	}
	return true
}

func (ae *ActionExecutor) prepareInhibitionAction(item ActionItem) bool {
	return true
}

func (ae *ActionExecutor) executePreparedAction(item ActionItem) bool {
	switch item.Type {
	case ActionOutput:
		return ae.executeOutputAction(item)
	case ActionToolExecution:
		return ae.executeToolAction(item)
	case ActionMemoryStore:
		return ae.executeMemoryAction(item)
	case ActionStateUpdate:
		return ae.executeStateAction(item)
	case ActionSocialResponse:
		return ae.executeSocialAction(item)
	case ActionInhibition:
		return ae.executeInhibitionAction(item)
	}
	return false
}

func (ae *ActionExecutor) executeOutputAction(item ActionItem) bool {
	// Emitir acción de salida
	payload, _ := json.Marshal(item.Payload)
	ae.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("out_%s", item.ID), Type: bus.Output,
		Source: "action_executor", Target: "output_sink",
		Priority: item.Priority, Timestamp: ae.clock.NowMilli(),
		Payload: payload, TTL: 1,
	})
	return true
}

func (ae *ActionExecutor) executeToolAction(item ActionItem) bool {
	// Emitir solicitud de herramienta
	payload, _ := json.Marshal(item.Payload)
	ae.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("tool_%s", item.ID), Type: bus.ToolRequest,
		Source: "action_executor", Target: "tool_decider",
		Priority: item.Priority, Timestamp: ae.clock.NowMilli(),
		Payload: payload, Tags: []string{"tool_request"}, TTL: 5,
	})
	return true
}

func (ae *ActionExecutor) executeMemoryAction(item ActionItem) bool {
	payload, _ := json.Marshal(item.Payload)
	ae.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("mem_%s", item.ID), Type: bus.Memory,
		Source: "action_executor", Target: "long_term_memory",
		Priority: 60, Timestamp: ae.clock.NowMilli(),
		Payload: payload, TTL: 5,
	})
	return true
}

func (ae *ActionExecutor) executeStateAction(item ActionItem) bool {
	payload, _ := json.Marshal(item.Payload)
	ae.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("state_%s", item.ID), Type: bus.Meta,
		Source: "action_executor", Target: "state_register",
		Priority: 50, Timestamp: ae.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
	return true
}

func (ae *ActionExecutor) executeSocialAction(item ActionItem) bool {
	payload, _ := json.Marshal(item.Payload)
	ae.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("social_%s", item.ID), Type: bus.Action,
		Source: "action_executor", Target: "output_formatter",
		Priority: 85, Timestamp: ae.clock.NowMilli(),
		Payload: payload, TTL: 3,
	})
	return true
}

func (ae *ActionExecutor) executeInhibitionAction(item ActionItem) bool {
	// Enviar señal inhibitoria
	payload, _ := json.Marshal(map[string]any{
		"inhibit":  true,
		"source":   item.Payload["source"],
		"reason":   item.Payload["reason"],
		"priority": item.Priority,
	})
	ae.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("inhib_%s", item.ID), Type: bus.Meta,
		Source: "action_executor", Target: "inhibitory_control",
		Priority: 95, Timestamp: ae.clock.NowMilli(),
		Payload: payload, TTL: 1,
	})
	return true
}

func (ae *ActionExecutor) enqueueAction(item ActionItem) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	// Verificar capacidad
	if len(ae.actionQueue) >= ae.maxQueue {
		// Remover la acción de menor prioridad
		lowestIdx := 0
		lowestPri := ae.actionQueue[0].Priority
		for i, action := range ae.actionQueue {
			if action.Priority < lowestPri {
				lowestPri = action.Priority
				lowestIdx = i
			}
		}
		ae.actionQueue = append(ae.actionQueue[:lowestIdx], ae.actionQueue[lowestIdx+1:]...)
	}

	ae.actionQueue = append(ae.actionQueue, item)
}

func (ae *ActionExecutor) flushActionQueue() {
	ae.mu.Lock()
	queue := make([]ActionItem, len(ae.actionQueue))
	copy(queue, ae.actionQueue)
	ae.actionQueue = ae.actionQueue[:0]
	ae.mu.Unlock()

	for _, item := range queue {
		pkt := bus.CognitivePacket{
			ID:       item.ID,
			Type:     bus.Action,
			Source:   "action_executor",
			Priority: item.Priority,
			Timestamp: ae.clock.NowMilli(),
		}
		ae.executeAction(item, pkt)
	}
}

func (ae *ActionExecutor) recordResult(item ActionItem, success bool, latency int64) {
	result := ActionResult{
		ActionID:  item.ID,
		Type:      item.Type,
		Success:   success,
		Latency:   latency,
		Timestamp: ae.clock.NowMilli(),
	}

	ae.mu.Lock()
	ae.execHistory = append(ae.execHistory, result)
	if len(ae.execHistory) > ae.maxHistory {
		ae.execHistory = ae.execHistory[1:]
	}
	ae.mu.Unlock()

	ae.updateStats(result)
}

func (ae *ActionExecutor) recordFailure(item ActionItem, reason string) {
	result := ActionResult{
		ActionID:  item.ID,
		Type:      item.Type,
		Success:   false,
		Error:     reason,
		Timestamp: ae.clock.NowMilli(),
	}

	ae.mu.Lock()
	ae.execHistory = append(ae.execHistory, result)
	if len(ae.execHistory) > ae.maxHistory {
		ae.execHistory = ae.execHistory[1:]
	}
	ae.mu.Unlock()

	ae.updateStats(result)
}

func (ae *ActionExecutor) emitActionResult(item ActionItem, success bool, pkt bus.CognitivePacket) {
	payload, _ := json.Marshal(map[string]any{
		"action_id": item.ID,
		"type":      item.Type.String(),
		"success":   success,
		"priority":  item.Priority,
	})

	ae.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("act_result_%s", item.ID), Type: bus.Meta,
		Source: "action_executor", Target: "meta_cognition",
		Priority: 30, Timestamp: ae.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (ae *ActionExecutor) updateStats(result ActionResult) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ae.stats.TotalActions++
	if result.Success {
		ae.stats.Executed++
	} else {
		ae.stats.Failed++
	}

	// Calcular latencia promedio
	if ae.stats.TotalActions > 0 {
		ae.stats.AvgLatency = float64(result.Latency) * 0.1 + ae.stats.AvgLatency*0.9
	}
}

func (ae *ActionExecutor) GetStats() ActionStats {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	stats := ae.stats
	stats.QueueDepth = len(ae.actionQueue)
	if ae.stats.TotalActions > 0 {
		stats.RetryRate = float64(ae.stats.Failed) / float64(ae.stats.TotalActions)
	}
	return stats
}

// ProcessNextAction procesa la siguiente acción en la cola
func (ae *ActionExecutor) ProcessNextAction() bool {
	ae.mu.Lock()
	if len(ae.actionQueue) == 0 {
		ae.mu.Unlock()
		return false
	}

	// Obtener la acción de mayor prioridad
	highestIdx := 0
	highestPri := ae.actionQueue[0].Priority
	for i, action := range ae.actionQueue {
		if action.Priority > highestPri {
			highestPri = action.Priority
			highestIdx = i
		}
	}

	item := ae.actionQueue[highestIdx]
	ae.actionQueue = append(ae.actionQueue[:highestIdx], ae.actionQueue[highestIdx+1:]...)
	ae.mu.Unlock()

	// Ejecutar
	pkt := bus.CognitivePacket{
		ID:       item.ID,
		Type:     bus.Action,
		Source:   "action_executor",
		Priority: item.Priority,
		Timestamp: ae.clock.NowMilli(),
	}
	ae.executeAction(item, pkt)

	return true
}

// RetardAction implementa el retardo voluntario de respuestas
func (ae *ActionExecutor) RetardAction(item ActionItem, duration time.Duration) {
	item.ExecuteAt = ae.clock.NowMilli() + duration.Milliseconds()
	ae.enqueueAction(item)
}
