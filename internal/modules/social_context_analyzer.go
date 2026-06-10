package modules

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/memory"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type SocialValidation struct {
	CoherenceScore   float64
	IntentionDetected string
	EmotionalState  string
	TrustLevel      float64
}

type SocialContextAnalyzer struct {
	sched    *scheduler.Scheduler
	clock    scheduler.Clock
	stateReg *StateRegister
	mem      *memory.MemoryBus
	mu       sync.Mutex
	agentProfiles map[string]*AgentProfile
}

type AgentProfile struct {
	AgentID             string
	Name                string
	RelationshipType    string
	Familiarity         float64
	TrustScore          float64
	EmotionalValence    float64
	LastInteraction     int64
	InteractionCount    int
	CommunicationStyle  string
	PredictedState      string
	Inconsistencies     int
}

func NewSocialContextAnalyzer(stateReg *StateRegister, clock scheduler.Clock, mem *memory.MemoryBus) *SocialContextAnalyzer {
	return &SocialContextAnalyzer{
		stateReg: stateReg, clock: clock, mem: mem,
		agentProfiles: make(map[string]*AgentProfile),
	}
}

func (sca *SocialContextAnalyzer) SetScheduler(s *scheduler.Scheduler) { sca.sched = s }

func (sca *SocialContextAnalyzer) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }
	thought, err := pkt.AsThoughtState()
	if err != nil { return }
	validation := sca.validateSocialContext(thought)
	validationJSON, _ := json.Marshal(validation)
	sca.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("social_%s", pkt.ID),
		Type:      bus.Meta,
		Source:    "social_context_analyzer",
		Target:    "control_planner",
		Priority:  65,
		Timestamp: sca.clock.NowMilli(),
		Payload:   validationJSON,
		Tags:      []string{"social_validation"},
		TTL:       3,
	})
}

func (sca *SocialContextAnalyzer) validateSocialContext(thought bus.ThoughtState) SocialValidation {
	state := sca.stateReg.GetState()
	coherence := 0.5
	if thought.IsSocial() { coherence += 0.2 }
	if thought.IsFromKnownAgent() { coherence += 0.15 }
	if state.PresionSocial > 0.5 { coherence += 0.1 }
	return SocialValidation{
		CoherenceScore:   clamp(coherence, 0, 1),
		IntentionDetected: sca.detectIntention(thought),
		EmotionalState:  sca.classifyEmotion(state),
		TrustLevel:      sca.getAgentTrust(thought.Source),
	}
}

func (sca *SocialContextAnalyzer) detectIntention(thought bus.ThoughtState) string {
	if thought.IsUrgent() { return "urgency" }
	if thought.IsSocial() { return "social_connection" }
	if thought.IsQuestion() { return "information_seeking" }
	if thought.HasImplicitContent() { return "implicit_communication" }
	return "neutral"
}

func (sca *SocialContextAnalyzer) classifyEmotion(state SystemState) string {
	if state.Valencia > 0.6 && state.Intensidad > 0.5 { return "positive_high_arousal" }
	if state.Valencia > 0.6 && state.Intensidad <= 0.5 { return "positive_low_arousal" }
	if state.Valencia < 0.4 && state.Intensidad > 0.5 { return "negative_high_arousal" }
	if state.Valencia < 0.4 && state.Intensidad <= 0.5 { return "negative_low_arousal" }
	return "neutral"
}

func (sca *SocialContextAnalyzer) getAgentTrust(agentID string) float64 {
	if agentID == "" || agentID == "unknown" { return 0.5 }
	sca.mu.Lock()
	defer sca.mu.Unlock()
	if profile, exists := sca.agentProfiles[agentID]; exists {
		return profile.TrustScore
	}
	return 0.5
}

func (sca *SocialContextAnalyzer) extractTags(payload string) []string {
	l := strings.ToLower(payload)
	t := []string{"tagged"}
	for _, p := range []string{"hola", "buenas", "hey", "adiós", "gracias", "por favor", "disculpa"} {
		if strings.Contains(l, p) { t = append(t, "social"); break }
	}
	for _, p := range []string{"urgente", "ahora", "ya", "inmediato", "crítico", "emergencia"} {
		if strings.Contains(l, p) { t = append(t, "urgent"); break }
	}
	if strings.Contains(l, "?") {
		t = append(t, "question")
	}
	for _, p := range []string{"batería", "battery", "ubicación", "gps", "wifi", "cámara", "foto", "linterna", "notificación", "alarma", "configuración"} {
		if strings.Contains(l, p) { t = append(t, "tool_request"); break }
	}
	return t
}
