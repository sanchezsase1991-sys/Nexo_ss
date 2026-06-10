package scheduler

import "github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"

type FairnessController interface {
	ApplyFairness(packets []bus.CognitivePacket, now int64)
}

type DefaultFairnessController struct{ windowSize int }

func NewDefaultFairnessController(windowSize int) *DefaultFairnessController {
	return &DefaultFairnessController{windowSize: windowSize}
}

func (fc *DefaultFairnessController) ApplyFairness(packets []bus.CognitivePacket, now int64) {
	for i := range packets {
		ticksWaiting := (now - packets[i].EnqueuedAt) / 10
		if ticksWaiting > 50 {
			packets[i].FairnessBoost = int(ticksWaiting / 10)
		}
	}
}
