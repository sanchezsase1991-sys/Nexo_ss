package modules

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/memory"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type SocialContextAnalyzer struct {
	sched    *scheduler.Scheduler
	clock    scheduler.Clock
	stateReg *StateRegister
	agents   map[bus.AgentID]bus.AgentProfile
	mu       sync.RWMutex
	mem      *memory.MemoryBus
}

func NewSocialContextAnalyzer(stateReg *StateRegister, clock scheduler.Clock, mem *memory.MemoryBus) *SocialContextAnalyzer {
	return &SocialContextAnalyzer{
		stateReg: stateReg, clock: clock, mem: mem,
		agents: make(map[bus.AgentID]bus.AgentProfile),
	}
}

func (sca *SocialContextAnalyzer) SetScheduler(s *scheduler.Scheduler) { sca.sched = s }

func (sca *SocialContextAnalyzer) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }
	thought, err := pkt.AsThoughtState()
	if err != nil { return }
	validation := sca.validateSocialContext(thought)
	agentID := bus.AgentID(thought.Source)
	sca.updateAgentModel(agentID, thought, validation)
	rp, _ := json.Marshal(map[string]interface{}{"thought_id": thought.OriginalID, "validation": validation})
	sca.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("social_%s", pkt.ID), Type: bus.Thought, Source: "social_context_analyzer",
		Target: "control_planner", Priority: 75, Timestamp: sca.clock.NowMilli(),
		Payload: rp, Tags: []string{"social_context"}, TTL: 3,
	})
	if validation.ShouldInhibit {
		sca.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("inhibit_%s", pkt.ID), Type: bus.Action, Source: "social_context_analyzer",
			Target: "auto_response_regulator", Priority: 90, Timestamp: sca.clock.NowMilli(),
			Payload: []byte(fmt.Sprintf(`{"reason":"%s"}`, validation.InhibitReason)),
			Tags: []string{"inhibited"}, TTL: 2,
		})
	}
}

func (sca *SocialContextAnalyzer) validateSocialContext(thought bus.ThoughtState) bus.ValidationResult {
	state := sca.stateReg.GetState()
	coherence := 0.8
	if thought.HasImplicitContent() && state.Saturacion > 0.7 { coherence -= 0.3 }
	if thought.IsFromKnownAgent() {
		agentID := bus.AgentID(thought.Source)
		sca.mu.RLock()
		profile, ok := sca.agents[agentID]
		sca.mu.RUnlock()
		if ok { coherence = clamp(coherence+profile.TrustScore*0.2, 0, 1) }
	}
	socialRisk := 0.1
	if thought.Intensity > 0.8 && !thought.IsFromKnownAgent() { socialRisk += 0.3 }
	if thought.IsUrgent() && state.Saturacion > 0.6 { socialRisk += 0.2 }
	shouldInhibit := false
	inhibitReason := ""
	if socialRisk > 0.7 { shouldInhibit = true; inhibitReason = "high_social_risk" }
	if coherence < 0.3 { shouldInhibit = true; inhibitReason = "low_coherence" }
	optimalTone := "balanced"
	if thought.IsSocial() && state.Valencia > 0.5 { optimalTone = "empathic" }
	if thought.IsUrgent() { optimalTone = "direct" }
	if state.Saturacion > 0.7 { optimalTone = "minimal" }
	return bus.ValidationResult{
		CoherenceScore: coherence, ShouldInhibit: shouldInhibit,
		InhibitReason: inhibitReason, OptimalTone: optimalTone, SocialRisk: socialRisk,
	}
}

func (sca *SocialContextAnalyzer) updateAgentModel(agentID bus.AgentID, thought bus.ThoughtState, validation bus.ValidationResult) {
	sca.mu.Lock()
	defer sca.mu.Unlock()
	profile, exists := sca.agents[agentID]
	if !exists { profile = bus.AgentProfile{ID: agentID, TrustScore: 0.5, RelationshipType: "stranger"} }
	profile.LastInteraction = sca.clock.NowMilli()
	profile.InteractionCount++
	profile.TrustScore = clamp(profile.TrustScore+0.01, 0, 1)
	sca.agents[agentID] = profile
}
