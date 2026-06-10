package modules

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type AttentionStateStr string

const (
	AttentionStateFlowStr      AttentionStateStr = "flow"
	AttentionStateSustainedStr AttentionStateStr = "sustained"
	AttentionStateDegradingStr AttentionStateStr = "degrading"
	AttentionStateCollapsedStr AttentionStateStr = "collapsed"
)

type AttentionController struct {
	sched           *scheduler.Scheduler
	clock           scheduler.Clock
	stateReg        *StateRegister
	mu              sync.Mutex
	focusStart      int64
	currentState    AttentionStateStr
	purposeThreshold float64
}

func NewAttentionController(stateReg *StateRegister, clock scheduler.Clock) *AttentionController {
	return &AttentionController{
		stateReg:         stateReg,
		clock:            clock,
		currentState:     AttentionStateSustainedStr,
		purposeThreshold: 0.6,
	}
}

func (ac *AttentionController) SetScheduler(s *scheduler.Scheduler) { ac.sched = s }

func (ac *AttentionController) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Meta { return }
	var payload map[string]interface{}
	json.Unmarshal(pkt.Payload, &payload)
	if event, ok := payload["attention_event"].(string); ok {
		switch event {
		case "focus_start":
			ac.focusStart = ac.clock.NowMilli()
			ac.currentState = AttentionStateSustainedStr
			log.Printf("[Attention] Focus started")
		case "focus_end":
			elapsed := float64(ac.clock.NowMilli()-ac.focusStart) / 60000.0
			log.Printf("[Attention] Focus ended after %.1f minutes", elapsed)
		}
	}
}

func (ac *AttentionController) MonitorAttention() {
	state := ac.stateReg.GetState()
	purposeSignal := state.Valencia*0.4 + state.Motivacion*0.3 + state.PresionSocial*0.3
	capacity := state.CognitiveCapacity()
	switch {
	case purposeSignal > 0.7 && capacity > 0.6:
		ac.currentState = AttentionStateFlowStr
	case purposeSignal > 0.4 && capacity > 0.4:
		ac.currentState = AttentionStateSustainedStr
	case capacity > 0.2:
		ac.currentState = AttentionStateDegradingStr
		ac.emitFatigueWarning()
	default:
		ac.currentState = AttentionStateCollapsedStr
		ac.emitCollapseAlert()
	}
	log.Printf("[Attention] State: %s (purpose=%.2f, capacity=%.2f)", ac.currentState, purposeSignal, capacity)
}

func (ac *AttentionController) emitFatigueWarning() {
	ac.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("attn_fatigue_%d", ac.clock.NowMilli()), Type: bus.Meta,
		Source: "attention_controller", Target: "control_planner", Priority: 70,
		Timestamp: ac.clock.NowMilli(),
		Payload:   []byte(`{"alert":"fatiga_atención_inminente","recomendacion":"modo_baja_carga"}`),
		Tags:      []string{"attention", "fatigue"}, TTL: 3,
	})
}

func (ac *AttentionController) emitCollapseAlert() {
	ac.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("attn_collapse_%d", ac.clock.NowMilli()), Type: bus.Meta,
		Source: "attention_controller", Target: "control_planner", Priority: 95,
		Timestamp: ac.clock.NowMilli(),
		Payload:   []byte(`{"alert":"colapso_atencional","recomendacion":"modo_seguro"}`),
		Tags:      []string{"attention", "collapse"}, TTL: 3,
	})
}

func (ac *AttentionController) StartMonitoring() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C { ac.MonitorAttention() }
	}()
}
