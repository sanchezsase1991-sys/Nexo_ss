package modules

import (
	"fmt"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type InputAcquisition struct {
	sched *scheduler.Scheduler
	clock scheduler.Clock
}

func NewInputAcquisition(clock scheduler.Clock) *InputAcquisition { return &InputAcquisition{clock: clock} }
func (i *InputAcquisition) SetScheduler(s *scheduler.Scheduler)    { i.sched = s }

func (i *InputAcquisition) ReceiveSignal(payload, source string, intensity float64, tags []string) {
	i.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("in_%d", i.clock.NowMilli()), Type: bus.Perception, Source: source,
		Target: "signal_tagger", Priority: 80, Timestamp: i.clock.NowMilli(),
		Payload: []byte(payload), Tags: append(tags, fmt.Sprintf("intensity:%.2f", intensity)), TTL: 10,
	})
}

func (i *InputAcquisition) Handle(pkt bus.CognitivePacket) {}
