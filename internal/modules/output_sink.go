package modules

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type OutputSink struct {
	LastResponse string
}

func NewOutputSink() *OutputSink { return &OutputSink{} }
func (o *OutputSink) SetScheduler(s *scheduler.Scheduler) {}

func (o *OutputSink) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Output { return }
	var out map[string]any
	json.Unmarshal(pkt.Payload, &out)
	msg, _ := out["message"].(string)
	if msg == "" { return }
	clean := strings.TrimPrefix(msg, "[CRUDO] ")
	clean = strings.TrimPrefix(clean, "[COLAPSO] ")
	clean = strings.TrimPrefix(clean, "[DEGRADADO] ")
	clean = strings.TrimSpace(clean)
	if clean == "" { return }
	o.LastResponse = clean
	fmt.Printf("╭─ Nexo ──────────────────────────────\n│ %s\n╰────────────────────────────────\n", clean)
}
