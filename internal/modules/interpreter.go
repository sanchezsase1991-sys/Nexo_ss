package modules

import "github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"

type Interpreter interface {
	GenerateMessage(thought bus.ThoughtState, decision string, score float64) string
	SelectTone(state SystemState, thought bus.ThoughtState) string
}

type DefaultInterpreter struct{}

func NewDefaultInterpreter() *DefaultInterpreter { return &DefaultInterpreter{} }

func (di *DefaultInterpreter) GenerateMessage(thought bus.ThoughtState, decision string, score float64) string {
	switch {
	case thought.IsUrgent() && thought.IsQuestion(): return "Entendido. Procesando tu solicitud urgente."
	case thought.IsUrgent(): return "Recibido. Atendiendo con urgencia."
	case thought.IsQuestion(): return "Procesando tu consulta."
	case thought.IsSocial(): return "Hola. ¿En qué puedo ayudarte?"
	case thought.IsToolRequest(): return "Activando herramienta solicitada."
	default: return "Solicitud procesada."
	}
}

func (di *DefaultInterpreter) SelectTone(state SystemState, thought bus.ThoughtState) string {
	if state.Stress() > 0.5 { return "direct" }
	if state.Valencia > 0.6 { return "empathic" }
	if state.Saturacion > 0.7 { return "minimal" }
	if thought.Tier == bus.TierHigh { return "urgent" }
	return "balanced"
}
