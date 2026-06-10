package modules

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type SystemDiagnostic struct {
	Timestamp         int64
	Saturacion        float64
	Intensidad        float64
	Valencia          float64
	Stress            float64
	CognitiveCapacity float64
	AttentionMode     string
	Recommendation    string
	Overflow          bool
	FalsePositives    int
	BugsDetected      int
	InhibitCount      int
}

type MetaCognitionMonitor struct {
	sched          *scheduler.Scheduler
	clock          scheduler.Clock
	stateReg       *StateRegister
	wm             *WorkingMemoryManager
	arr            *AutoResponseRegulator
	ltm            *LongTermMemory
	diagnostics    []SystemDiagnostic
	mu             sync.RWMutex
	bugsDetected   int
	falsePositives int
}

func NewMetaCognitionMonitor(stateReg *StateRegister, wm *WorkingMemoryManager, clock scheduler.Clock) *MetaCognitionMonitor {
	return &MetaCognitionMonitor{
		stateReg: stateReg, wm: wm, clock: clock,
		diagnostics: make([]SystemDiagnostic, 0, 100),
	}
}

func (mcm *MetaCognitionMonitor) SetScheduler(s *scheduler.Scheduler)              { mcm.sched = s }
func (mcm *MetaCognitionMonitor) SetAutoResponseRegulator(arr *AutoResponseRegulator) { mcm.arr = arr }
func (mcm *MetaCognitionMonitor) SetLongTermMemory(ltm *LongTermMemory)              { mcm.ltm = ltm }

func (mcm *MetaCognitionMonitor) Handle(pkt bus.CognitivePacket) {
	if pkt.Type == bus.Meta {
		var payload map[string]interface{}
		json.Unmarshal(pkt.Payload, &payload)
		if _, ok := payload["error"]; ok { mcm.mu.Lock(); mcm.bugsDetected++; mcm.mu.Unlock() }
	}
}

func (mcm *MetaCognitionMonitor) RunDiagnostic() {
	state := mcm.stateReg.GetState()
	diag := SystemDiagnostic{
		Timestamp: mcm.clock.NowMilli(), Saturacion: state.Saturacion,
		Intensidad: state.Intensidad, Valencia: state.Valencia,
		Stress: state.Stress(), CognitiveCapacity: state.CognitiveCapacity(),
	}
	if mcm.wm != nil { impact := mcm.wm.CheckOverflow(); diag.Overflow = impact.CalibrationPrecision < 0.8 }
	if mcm.arr != nil { stats := mcm.arr.GetStats(); diag.InhibitCount = stats["total_inhibitions"] }
	mcm.mu.RLock()
	diag.FalsePositives = mcm.falsePositives
	diag.BugsDetected = mcm.bugsDetected
	mcm.mu.RUnlock()
	diag.Recommendation = mcm.generateRecommendation(diag)
	mcm.mu.Lock()
	mcm.diagnostics = append(mcm.diagnostics, diag)
	if len(mcm.diagnostics) > 100 { mcm.diagnostics = mcm.diagnostics[1:] }
	mcm.mu.Unlock()
	diagJSON, _ := json.Marshal(diag)
	mcm.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("diag_%d", mcm.clock.NowMilli()), Type: bus.Meta,
		Source: "meta_cognition_monitor", Target: "control_planner", Priority: 35,
		Timestamp: mcm.clock.NowMilli(), Payload: diagJSON, Tags: []string{"diagnostic"}, TTL: 2,
	})
	if diag.Overflow {
		mcm.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("overflow_%d", mcm.clock.NowMilli()), Type: bus.Meta,
			Source: "meta_cognition_monitor", Target: "state_register", Priority: 85,
			Timestamp: mcm.clock.NowMilli(), Payload: []byte(`{"saturacion":0.85}`), TTL: 2,
		})
	}
}

func (mcm *MetaCognitionMonitor) generateRecommendation(diag SystemDiagnostic) string {
	switch {
	case diag.Overflow: return "CRÍTICO: Memoria de trabajo saturada."
	case diag.Stress > 0.7: return "ALTO: Nivel de estrés elevado."
	case diag.CognitiveCapacity < 0.3: return "MEDIO: Capacidad cognitiva baja."
	default: return "Sistema operando dentro de parámetros normales."
	}
}

func (mcm *MetaCognitionMonitor) StartMonitoring() {
	go func() {
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for range ticker.C { mcm.RunDiagnostic() }
	}()
}
