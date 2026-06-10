package modules

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// OutputMonitor monitorea la salida verbal contra la intención comunicativa
type OutputMonitor struct {
	sched          *scheduler.Scheduler
	clock          scheduler.Clock
	stateReg       *StateRegister
	mu             sync.Mutex
	corrections    int
	discrepancies  int
}

func NewOutputMonitor(stateReg *StateRegister, clock scheduler.Clock) *OutputMonitor {
	return &OutputMonitor{
		stateReg: stateReg,
		clock:    clock,
	}
}

func (om *OutputMonitor) SetScheduler(s *scheduler.Scheduler) { om.sched = s }

// Handle monitorea la salida y detecta discrepancias
func (om *OutputMonitor) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Action && pkt.Type != bus.Output { return }

	var payload map[string]interface{}
	json.Unmarshal(pkt.Payload, &payload)

	message, ok := payload["message"].(string)
	if !ok { return }

	intention, _ := payload["intention"].(string)

	// Comparar salida con intención
	if intention != "" {
		discrepancy := om.detectDiscrepancy(message, intention)
		if discrepancy != "" {
			om.reportDiscrepancy(message, intention, discrepancy)
		}
	}

	// Monitorear calibración según estado
	state := om.stateReg.GetState()
	if state.Saturacion > 0.7 {
		om.reportCalibrationDegradation(message, state)
	}
}

func (om *OutputMonitor) detectDiscrepancy(message, intention string) string {
	// Detectar si el mensaje no refleja la intención
	if strings.Contains(intention, "empathic") && !strings.Contains(message, "?") {
		return "posible_falta_de_empatía"
	}
	if strings.Contains(intention, "direct") && len(message) > 200 {
		return "mensaje_demasiado_largo_para_intención_directa"
	}
	return ""
}

func (om *OutputMonitor) reportDiscrepancy(message, intention, discrepancy string) {
	om.mu.Lock()
	om.discrepancies++
	count := om.discrepancies
	om.mu.Unlock()

	om.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("discrepancy_%d", om.clock.NowMilli()),
		Type:      bus.Meta,
		Source:    "output_monitor",
		Target:    "control_planner",
		Priority:  55,
		Timestamp: om.clock.NowMilli(),
		Payload:   []byte(fmt.Sprintf(`{"discrepancy":"%s","count":%d,"intention":"%s"}`, discrepancy, count, intention)),
		Tags:      []string{"output_discrepancy"},
		TTL:       3,
	})
}

func (om *OutputMonitor) reportCalibrationDegradation(message string, state SystemState) {
	om.mu.Lock()
	om.corrections++
	corrections := om.corrections
	om.mu.Unlock()

	om.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("calib_degrad_%d", om.clock.NowMilli()),
		Type:      bus.Meta,
		Source:    "output_monitor",
		Target:    "output_formatter",
		Priority:  50,
		Timestamp: om.clock.NowMilli(),
		Payload:   []byte(fmt.Sprintf(`{"alert":"calibration_degraded","saturation":%.2f,"corrections":%d}`, state.Saturacion, corrections)),
		Tags:      []string{"calibration", "degradation"},
		TTL:       2,
	})
}

func (om *OutputMonitor) GetStats() map[string]int {
	om.mu.Lock()
	defer om.mu.Unlock()
	return map[string]int{
		"corrections":   om.corrections,
		"discrepancies": om.discrepancies,
	}
}
