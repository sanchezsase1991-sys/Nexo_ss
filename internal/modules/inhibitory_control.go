package modules

import (
	"fmt"
	"sync"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// PendingResponse representa una respuesta en buffer de salida
type PendingResponse struct {
	ID           string
	Content      string
	Timestamp    int64
	Intensity    float64
	ContextScore float64
}

// InhibitedThread representa un hilo inhibido
type InhibitedThread struct {
	ID          string
	Original    PendingResponse
	Reason      string
	InhibitedAt int64
	Cost        float64
}

// InhibitoryControl implementa el control inhibitorio
type InhibitoryControl struct {
	sched             *scheduler.Scheduler
	clock             scheduler.Clock
	stateReg          *StateRegister
	socialAnalyzer    *SocialContextAnalyzer
	mu                sync.RWMutex
	responseBuffer    []PendingResponse
	inhibitedThreads  map[string]InhibitedThread
	inhibitionCost    float64
	maxBufferCapacity int
	totalInhibitions  int
}

func NewInhibitoryControl(stateReg *StateRegister, social *SocialContextAnalyzer, clock scheduler.Clock) *InhibitoryControl {
	return &InhibitoryControl{
		stateReg:          stateReg,
		socialAnalyzer:    social,
		clock:             clock,
		responseBuffer:    make([]PendingResponse, 0, 100),
		inhibitedThreads:  make(map[string]InhibitedThread),
		maxBufferCapacity: 100,
	}
}

func (ic *InhibitoryControl) SetScheduler(s *scheduler.Scheduler) { ic.sched = s }

// Handle procesa señales para control inhibitorio
func (ic *InhibitoryControl) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }

	thought, err := pkt.AsThoughtState()
	if err != nil { return }

	state := ic.stateReg.GetState()

	// Generar respuesta preparatoria
	response := PendingResponse{
		ID:        fmt.Sprintf("resp_%s", pkt.ID),
		Content:   thought.Payload,
		Timestamp: ic.clock.NowMilli(),
		Intensity: thought.Intensity,
	}

	// Evaluar contexto social en paralelo
	contextScore := ic.evaluateContext(thought)

	// Calcular umbral óptimo según estado
	optimalThreshold := ic.calculateOptimalThreshold(state)

	// Decidir inhibición
	if contextScore < optimalThreshold {
		ic.inhibit(response, fmt.Sprintf("context_score_below_threshold_%.2f", optimalThreshold), state)
		return
	}

	// Retardo voluntario — mantener en buffer y simular
	if thought.Intensity > 0.6 && state.Saturacion < 0.75 {
		ic.voluntaryDelay(response, thought, state)
		return
	}

	// Liberar respuesta
	ic.release(response)
}

func (ic *InhibitoryControl) evaluateContext(thought bus.ThoughtState) float64 {
	if ic.socialAnalyzer == nil { return 0.5 }
	validation := ic.socialAnalyzer.validateSocialContext(thought)
	return validation.CoherenceScore
}

func (ic *InhibitoryControl) calculateOptimalThreshold(state SystemState) float64 {
	threshold := 0.5
	if state.Saturacion > 0.7 { threshold += 0.15 }
	if state.Intensidad > 0.7 { threshold += 0.1 }
	return threshold
}

// inhibit suprime una respuesta generando micro-latencia y costo
func (ic *InhibitoryControl) inhibit(response PendingResponse, reason string, state SystemState) {
	start := ic.clock.NowMilli()

	ic.mu.Lock()
	ic.totalInhibitions++
	ic.inhibitedThreads[response.ID] = InhibitedThread{
		ID:          response.ID,
		Original:    response,
		Reason:      reason,
		InhibitedAt: start,
		Cost:        0.05,
	}
	ic.mu.Unlock()

	latency := ic.clock.NowMilli() - start

	// Emitir costo de inhibición al StateRegister
	ic.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("inhibit_cost_%s", response.ID),
		Type:      bus.Meta,
		Source:    "inhibitory_control",
		Target:    "state_register",
		Priority:  40,
		Timestamp: ic.clock.NowMilli(),
		Payload:   []byte(fmt.Sprintf(`{"saturacion_delta":0.03,"intensidad_delta":0.05,"latency_ms":%d}`, latency)),
		TTL:       2,
	})

	// Notificar a LTM para registro de inhibición
	ic.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("inhibit_log_%s", response.ID),
		Type:      bus.Meta,
		Source:    "inhibitory_control",
		Target:    "long_term_memory",
		Priority:  30,
		Timestamp: ic.clock.NowMilli(),
		Payload:   []byte(fmt.Sprintf(`{"event":"inhibition","reason":"%s","cost":0.05}`, reason)),
		TTL:       3,
	})
}

// voluntaryDelay implementa el retardo voluntario activo
func (ic *InhibitoryControl) voluntaryDelay(response PendingResponse, thought bus.ThoughtState, state SystemState) {
	ic.mu.Lock()
	ic.responseBuffer = append(ic.responseBuffer, response)
	if len(ic.responseBuffer) > ic.maxBufferCapacity {
		ic.responseBuffer = ic.responseBuffer[1:]
	}
	ic.mu.Unlock()

	// Solicitar simulación predictiva durante el retardo
	ic.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("delay_sim_%s", response.ID),
		Type:      bus.Thought,
		Source:    "inhibitory_control",
		Target:    "predictive_simulator",
		Priority:  60,
		Timestamp: ic.clock.NowMilli(),
		Payload:   []byte(thought.Payload),
		Tags:      []string{"voluntary_delay", "simulation_request"},
		TTL:       3,
	})

	// Programar liberación después del timeout adaptativo
	timeout := ic.calculateTimeout(state, thought)
	go func() {
		time.Sleep(timeout)
		ic.mu.Lock()
		// Verificar si la respuesta sigue en buffer
		for i, resp := range ic.responseBuffer {
			if resp.ID == response.ID {
				ic.responseBuffer = append(ic.responseBuffer[:i], ic.responseBuffer[i+1:]...)
				ic.mu.Unlock()
				ic.release(resp)
				return
			}
		}
		ic.mu.Unlock()
	}()
}

func (ic *InhibitoryControl) calculateTimeout(state SystemState, thought bus.ThoughtState) time.Duration {
	base := 800 * time.Millisecond
	if thought.IsSocial() && state.PresionSocial > 0.7 { base = 300 * time.Millisecond }
	if state.Saturacion > 0.6 { base = 200 * time.Millisecond }
	if thought.Intensity > 0.9 { base = 1200 * time.Millisecond }
	return base
}

func (ic *InhibitoryControl) release(response PendingResponse) {
	ic.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("released_%s", response.ID),
		Type:      bus.Action,
		Source:    "inhibitory_control",
		Target:    "output_formatter",
		Priority:  80,
		Timestamp: ic.clock.NowMilli(),
		Payload:   []byte(response.Content),
		Tags:      []string{"released"},
		TTL:       5,
	})
}

func (ic *InhibitoryControl) GetStats() map[string]int {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return map[string]int{
		"total_inhibitions":  ic.totalInhibitions,
		"buffer_size":        len(ic.responseBuffer),
		"inhibited_threads":  len(ic.inhibitedThreads),
	}
}
