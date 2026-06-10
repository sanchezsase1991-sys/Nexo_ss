package modules

import (
	"encoding/json"
	"fmt"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type ResourceEstimate struct {
	EstimatedTime float64
	ActualTime    float64
	EmotionalLoad float64
	SocialLoad    float64
}

type ResourceEstimator struct {
	sched    *scheduler.Scheduler
	clock    scheduler.Clock
	stateReg *StateRegister
}

func NewResourceEstimator(stateReg *StateRegister, clock scheduler.Clock) *ResourceEstimator {
	return &ResourceEstimator{stateReg: stateReg, clock: clock}
}

func (re *ResourceEstimator) SetScheduler(s *scheduler.Scheduler) { re.sched = s }

func (re *ResourceEstimator) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }
	thought, err := pkt.AsThoughtState()
	if err != nil { return }
	state := re.stateReg.GetState()
	baseTime := thought.Intensity * 1000
	emotionalCost := state.Stress() * 2.0
	socialCost := 0.0
	if thought.IsSocial() { socialCost = state.PresionSocial * 1.5 }
	estimate := ResourceEstimate{
		EstimatedTime: baseTime * 0.7,
		ActualTime:    baseTime,
		EmotionalLoad: emotionalCost,
		SocialLoad:    socialCost,
	}
	estimateJSON, _ := json.Marshal(estimate)
	re.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("resource_est_%d", re.clock.NowMilli()), Type: bus.Meta,
		Source: "resource_estimator", Target: "control_planner", Priority: 55,
		Timestamp: re.clock.NowMilli(), Payload: estimateJSON, TTL: 3,
	})
}
