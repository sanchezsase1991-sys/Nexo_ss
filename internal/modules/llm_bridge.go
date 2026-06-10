package modules

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/llm"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type LLMBridge struct {
	sched   *scheduler.Scheduler
	clock   scheduler.Clock
	pool    *llm.LLMPool
	enabled bool
}

func NewLLMBridge(pool *llm.LLMPool, clock scheduler.Clock) *LLMBridge {
	return &LLMBridge{pool: pool, clock: clock, enabled: pool != nil}
}

func (lb *LLMBridge) SetScheduler(s *scheduler.Scheduler) { lb.sched = s }
func (lb *LLMBridge) SetEnabled(enabled bool)              { lb.enabled = enabled }

func (lb *LLMBridge) Handle(pkt bus.CognitivePacket) {
	if !lb.enabled { return }
	if pkt.Type == bus.Thought && containsTagStr(pkt.Tags, "llm_request") { lb.processLLMRequest(pkt) }
}

func (lb *LLMBridge) processLLMRequest(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil { return }
	prompt := lb.buildPrompt(thought)
	response, err := lb.pool.Generate("reasoning", prompt, 512, 0.3)
	if err != nil { log.Printf("[LLMBridge] Generation failed: %v", err); return }
	result := map[string]interface{}{"thought_id": thought.OriginalID, "prompt": prompt, "response": response}
	payload, _ := json.Marshal(result)
	lb.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("llm_%s", pkt.ID), Type: bus.Thought, Source: "llm_bridge",
		Target: "control_planner", Priority: 75, Timestamp: lb.clock.NowMilli(),
		Payload: payload, Tags: []string{"llm_response"}, TTL: 5,
	})
}

func (lb *LLMBridge) buildPrompt(thought bus.ThoughtState) string {
	return fmt.Sprintf("[NEXO] Tipo:%s Intensidad:%.2f Payload:%s Tags:%v", thought.Tier, thought.Intensity, thought.Payload, thought.Tags)
}
