package bus

import "github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"

type Module interface {
	Handle(pkt CognitivePacket)
	SetScheduler(s *scheduler.Scheduler)
}
