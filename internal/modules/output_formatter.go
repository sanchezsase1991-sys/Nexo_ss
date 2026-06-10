package modules

import (
	"encoding/json"
	"fmt"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type OutputFormatter struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	interpreter Interpreter
}

func NewOutputFormatter(stateReg *StateRegister, clock scheduler.Clock, interpreter Interpreter) *OutputFormatter {
	return &OutputFormatter{stateReg: stateReg, clock: clock, interpreter: interpreter}
}

func (o *OutputFormatter) SetScheduler(s *scheduler.Scheduler) { o.sched = s }

func (o *OutputFormatter) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Action { return }
	var in map[string]any
	json.Unmarshal(pkt.Payload, &in)
	msg, _ := in["message"].(string)
	tone, _ := in["tone"].(string)
	degraded, _ := in["degraded_mode"].(bool)
	collapsed, _ := in["collapsed"].(bool)
	state := o.stateReg.GetState()
	out := map[string]any{"id": pkt.ID, "message": msg, "tone": tone, "ts": o.clock.NowMilli()}
	switch {
	case state.Saturacion > 0.85 || collapsed:
		out["tone"] = "collapse"
		out["message"] = "[COLAPSO] " + truncateStr(msg, 50)
	case state.Saturacion > 0.7:
		out["tone"] = "raw"
		out["message"] = "[CRUDO] " + msg
		out["calibration_lost"] = []string{"social_filter", "inhibitory_delay", "diplomatic_framing"}
	case state.Saturacion > 0.6:
		out["tone"] = "minimal"
		out["message"] = truncateStr(msg, 100)
	}
	if degraded { out["message"] = "[DEGRADADO] " + out["message"].(string) }
	b, _ := json.Marshal(out)
	o.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("out_%s", pkt.ID), Type: bus.Output, Source: "output_formatter",
		Target: "external", Priority: 95, Timestamp: o.clock.NowMilli(), Payload: b, TTL: 1,
	})
}
