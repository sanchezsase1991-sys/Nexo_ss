package modules

import (
	"fmt"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type Reframer struct {
	sched        *scheduler.Scheduler
	clock        scheduler.Clock
	stateReg     *StateRegister
	reframeCount int
}

func NewReframer(stateReg *StateRegister, clock scheduler.Clock) *Reframer {
	return &Reframer{stateReg: stateReg, clock: clock}
}

func (r *Reframer) SetScheduler(s *scheduler.Scheduler) { r.sched = s }

func (r *Reframer) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }
	_, err := pkt.AsThoughtState()
	if err != nil { return }
	state := r.stateReg.GetState()
	if state.Intensidad > 0.6 {
		intensityDelta := -state.Intensidad * 0.3
		r.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("reframe_%d", r.clock.NowMilli()), Type: bus.Meta,
			Source: "reframer", Target: "state_register", Priority: 70,
			Timestamp: r.clock.NowMilli(),
			Payload:   []byte(fmt.Sprintf(`{"intensidad_delta":%.4f}`, intensityDelta)),
			Tags:      []string{"reframe", "modulation"}, TTL: 2,
		})
		r.reframeCount++
	}
}

func (r *Reframer) GetReframeCount() int { return r.reframeCount }
