package modules

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type PrecompiledResponse struct {
	Patterns     []string
	Response     string
	Tone         string
	Confidence   float64
	UsageCount   int
	InhibitCount int
}

type AutoResponseRegulator struct {
	sched            *scheduler.Scheduler
	clock            scheduler.Clock
	stateReg         *StateRegister
	precompiled      []PrecompiledResponse
	mu               sync.RWMutex
	disonanceCount   int
	totalInhibitions int
}

func NewAutoResponseRegulator(stateReg *StateRegister, clock scheduler.Clock) *AutoResponseRegulator {
	return &AutoResponseRegulator{
		stateReg: stateReg, clock: clock,
		precompiled: []PrecompiledResponse{
			{Patterns: []string{"hola", "buenas", "hey"}, Response: "Hola, ¿en qué puedo ayudarte?", Tone: "warm_opening", Confidence: 0.95},
			{Patterns: []string{"adiós", "hasta luego", "nos vemos"}, Response: "Hasta luego.", Tone: "warm_closing", Confidence: 0.95},
			{Patterns: []string{"gracias"}, Response: "De nada.", Tone: "empathic", Confidence: 0.90},
			{Patterns: []string{"ayuda", "help"}, Response: "Puedo ayudarte con consultas y más.", Tone: "clear_helpful", Confidence: 0.90},
		},
	}
}

func (arr *AutoResponseRegulator) SetScheduler(s *scheduler.Scheduler) { arr.sched = s }

func (arr *AutoResponseRegulator) Handle(pkt bus.CognitivePacket) {
	if pkt.Type == bus.Action && containsTagStr(pkt.Tags, "inhibited") { arr.handleInhibition(pkt); return }
	if pkt.Type == bus.Thought { arr.checkPrecompiled(pkt) }
}

func (arr *AutoResponseRegulator) handleInhibition(pkt bus.CognitivePacket) {
	arr.mu.Lock()
	arr.totalInhibitions++
	arr.disonanceCount++
	arr.mu.Unlock()
	arr.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("arr_disonance_%d", arr.clock.NowMilli()), Type: bus.Meta,
		Source: "auto_response_regulator", Target: "state_register", Priority: 55,
		Timestamp: arr.clock.NowMilli(),
		Payload:   []byte(fmt.Sprintf(`{"intensidad_delta":0.05,"saturacion_delta":0.03,"disonance_count":%d}`, arr.disonanceCount)),
		TTL:       2,
	})
}

func (arr *AutoResponseRegulator) matchPrecompiled(payload string, pr PrecompiledResponse) bool {
	lower := strings.ToLower(payload)
	for _, p := range pr.Patterns { if strings.Contains(lower, p) { return true } }
	return false
}

func (arr *AutoResponseRegulator) checkPrecompiled(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil { return }
	state := arr.stateReg.GetState()
	if state.Saturacion > 0.8 { return }
	arr.mu.RLock()
	defer arr.mu.RUnlock()
	for _, pr := range arr.precompiled {
		if arr.matchPrecompiled(thought.Payload, pr) && pr.Confidence > 0.7 {
			payload, _ := json.Marshal(map[string]interface{}{"message": pr.Response, "tone": pr.Tone, "decision": "auto_response", "precompiled": true})
			arr.sched.Emit(bus.CognitivePacket{
				ID: fmt.Sprintf("auto_%s", pkt.ID), Type: bus.Action, Source: "auto_response_regulator",
				Target: "output_formatter", Priority: 80, Timestamp: arr.clock.NowMilli(),
				Payload: payload, Tags: []string{"auto_response"}, TTL: 3,
			})
			return
		}
	}
}

func (arr *AutoResponseRegulator) GetStats() map[string]int {
	arr.mu.RLock()
	defer arr.mu.RUnlock()
	return map[string]int{"disonance_count": arr.disonanceCount, "total_inhibitions": arr.totalInhibitions}
}
