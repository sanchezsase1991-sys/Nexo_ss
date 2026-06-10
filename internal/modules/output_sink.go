package modules

import (
	"encoding/json"
	"fmt"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type OutputSink struct{}

func NewOutputSink() *OutputSink                      { return &OutputSink{} }
func (o *OutputSink) SetScheduler(s *scheduler.Scheduler) {}

func (o *OutputSink) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Output { return }
	var out map[string]any
	json.Unmarshal(pkt.Payload, &out)
	if msg, ok := out["message"]; ok { fmt.Printf("[NEXO] %v\n", msg) } else { fmt.Printf("[NEXO] %s\n", string(pkt.Payload)) }
}
