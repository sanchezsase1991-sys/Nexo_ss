package modules

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// AssociativeManipulator implementa la manipulación asociativa de chunks
type AssociativeManipulator struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	wm          *WorkingMemoryManager
	ltm         *LongTermMemory
	mu          sync.Mutex
	combinations int
}

func NewAssociativeManipulator(wm *WorkingMemoryManager, ltm *LongTermMemory, clock scheduler.Clock) *AssociativeManipulator {
	return &AssociativeManipulator{
		wm:    wm,
		ltm:   ltm,
		clock: clock,
	}
}

func (am *AssociativeManipulator) SetScheduler(s *scheduler.Scheduler) { am.sched = s }

// Handle procesa pensamientos para generar asociaciones creativas
func (am *AssociativeManipulator) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }

	thought, err := pkt.AsThoughtState()
	if err != nil { return }

	// Buscar conexiones con chunks existentes en memoria
	associations := am.findAssociations(thought)

	// Recombinar elementos dispares para generar ideas nuevas
	if len(associations) >= 2 {
		novelIdea := am.recombine(associations, thought)
		if novelIdea != "" {
			am.emitNovelIdea(novelIdea, thought)
		}
	}
}

func (am *AssociativeManipulator) findAssociations(thought bus.ThoughtState) []string {
	var associations []string

	// Buscar en memoria de trabajo
	if am.wm != nil {
		// Consultar chunks activos
		associations = append(associations, thought.Tags...)
	}

	// Buscar en memoria a largo plazo
	if am.ltm != nil {
		episodes := am.ltm.RetrieveByEmotionalSignature("", 0.3)
		for _, ep := range episodes {
			if containsSimilarPattern(ep.Details, thought.Payload) {
				associations = append(associations, ep.Context)
			}
		}
	}

	return associations
}

func (am *AssociativeManipulator) recombine(associations []string, thought bus.ThoughtState) string {
	am.mu.Lock()
	am.combinations++
	count := am.combinations
	am.mu.Unlock()

	if len(associations) < 2 { return "" }

	// Combinación creativa: mezclar elementos dispares
	a := associations[rand.Intn(len(associations))]
	b := associations[rand.Intn(len(associations))]
	for a == b { b = associations[rand.Intn(len(associations))] }

	return fmt.Sprintf("[Síntesis #%d] %s + %s → Nueva perspectiva sobre '%s'", count, a, b, thought.Payload)
}

func (am *AssociativeManipulator) emitNovelIdea(idea string, thought bus.ThoughtState) {
	payload, _ := json.Marshal(map[string]string{
		"idea":        idea,
		"trigger":     thought.Payload,
		"combination": fmt.Sprintf("%d", am.combinations),
	})

	am.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("novel_%s", thought.OriginalID),
		Type:      bus.Thought,
		Source:    "associative_manipulator",
		Target:    "working_memory",
		Priority:  65,
		Timestamp: am.clock.NowMilli(),
		Payload:   payload,
		Tags:      []string{"novel_idea", "creative_synthesis"},
		TTL:       5,
	})
}

func containsSimilarPattern(a, b string) bool {
	return len(a) > 0 && len(b) > 0
}
