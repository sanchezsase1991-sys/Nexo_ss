package modules

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

const (
	maxToolHistory          = 200
	saturationBlockThreshold = 0.85
	minConfidenceThreshold   = 0.5
)

type ToolDecider struct {
	sched    *scheduler.Scheduler
	clock    scheduler.Clock
	stateReg *StateRegister
	registry map[bus.ToolName]bus.ToolCapability
	history  []bus.ToolResult
	mu       sync.RWMutex
}

func NewToolDecider(stateReg *StateRegister, clock scheduler.Clock) *ToolDecider {
	return &ToolDecider{
		stateReg: stateReg, clock: clock,
		registry: ToolRegistry,
		history:  make([]bus.ToolResult, 0, maxToolHistory),
	}
}

func (td *ToolDecider) SetScheduler(s *scheduler.Scheduler) { td.sched = s }

func (td *ToolDecider) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }
	thought, err := pkt.AsThoughtState()
	if err != nil { return }
	if !thought.IsToolRequest() { return }
	state := td.stateReg.GetState()
	if state.Saturacion > saturationBlockThreshold {
		log.Printf("[ToolDecider] Blocked by saturation %.2f", state.Saturacion)
		return
	}
	tool, confidence := td.matchTool(thought)
	if tool == nil { return }
	params := td.extractParams(thought, tool)
	if missing := td.missingRequired(tool, params); len(missing) > 0 {
		log.Printf("[ToolDecider] Missing params for %s: %v", tool.Name, missing)
		return
	}
	log.Printf("[ToolDecider] Matched tool: %s (confidence=%.2f)", tool.Name, confidence)
	result := td.executeTool(tool, params)
	td.mu.Lock()
	td.history = append(td.history, result)
	if len(td.history) > maxToolHistory { td.history = td.history[len(td.history)-maxToolHistory:] }
	td.mu.Unlock()
	td.emitToolResult(result, pkt)
}

func (td *ToolDecider) matchTool(thought bus.ThoughtState) (*bus.ToolCapability, float64) {
	payload := strings.ToLower(thought.Payload)
	var bestMatch *bus.ToolCapability
	bestScore := 0.0
	for _, tool := range td.registry {
		score := 0.0
		keywordMatchCount := 0
		for _, kw := range tool.Keywords {
			if strings.Contains(payload, kw) { score += 0.3; keywordMatchCount++ }
		}
		if strings.Contains(payload, strings.ToLower(string(tool.Name))) { score += 0.5 }
		if keywordMatchCount >= 2 { score += 0.2 }
		if score > bestScore { bestScore = score; t := tool; bestMatch = &t }
	}
	if bestScore >= minConfidenceThreshold { return bestMatch, clampF64(bestScore, 0.0, 1.0) }
	return nil, 0
}

func (td *ToolDecider) extractParams(thought bus.ThoughtState, tool *bus.ToolCapability) map[string]string {
	params := make(map[string]string)
	payload := thought.Payload
	for _, p := range tool.Params {
		extracted := td.extractParam(payload, p)
		if extracted != "" { params[p.Name] = extracted } else if p.Default != "" { params[p.Name] = p.Default }
	}
	return params
}

func (td *ToolDecider) extractParam(payload string, p bus.ToolParam) string {
	lower := strings.ToLower(payload)
	switch p.Type {
	case "int": return td.extractInt(lower, p.Name)
	case "string":
		switch p.Name {
		case "path": return td.extractAfterKeyword(payload, "path")
		case "command", "script", "expression": return td.extractAfterKeyword(lower, p.Name)
		case "url": return td.extractURL(payload)
		case "number": return td.extractPhoneNumber(lower)
		case "message", "text": return payload
		case "title": return "Nexo"
		case "uri": return td.extractURL(payload)
		default: return ""
		}
	default: if p.Default != "" { return p.Default }; return ""
	}
}

func (td *ToolDecider) extractInt(payload, paramName string) string {
	words := strings.Fields(payload)
	for i, w := range words {
		if strings.ToLower(w) == paramName && i+1 < len(words) {
			candidate := strings.TrimRight(words[i+1], ".,;:!?)")
			if isNumeric(candidate) { return candidate }
		}
	}
	for _, w := range words { clean := strings.TrimRight(w, ".,;:!?)"); if isNumeric(clean) { return clean } }
	return ""
}

func (td *ToolDecider) extractAfterKeyword(payload, keyword string) string {
	words := strings.Fields(payload)
	for i, w := range words { if strings.ToLower(w) == keyword && i+1 < len(words) { return strings.Join(words[i+1:], " ") } }
	return ""
}

func (td *ToolDecider) extractURL(payload string) string {
	words := strings.Fields(payload)
	for _, w := range words { clean := strings.TrimRight(w, ".,;:!?)"); if strings.HasPrefix(clean, "http") { return clean } }
	return ""
}

func (td *ToolDecider) extractPhoneNumber(payload string) string {
	words := strings.Fields(payload)
	for _, w := range words { clean := strings.TrimRight(w, ".,;:!?)"); if isNumeric(clean) && len(clean) >= 7 { return clean } }
	return ""
}

func (td *ToolDecider) missingRequired(tool *bus.ToolCapability, params map[string]string) []string {
	var missing []string
	for _, p := range tool.Params { if p.Required { if val, ok := params[p.Name]; !ok || strings.TrimSpace(val) == "" { missing = append(missing, p.Name) } } }
	return missing
}

func (td *ToolDecider) executeTool(tool *bus.ToolCapability, params map[string]string) bus.ToolResult {
	start := td.clock.NowMilli()
	result := bus.ToolResult{ToolName: tool.Name, Timestamp: start}
	switch tool.Name {
	case bus.ToolBattery: result.Success = true; result.Data = "Batería: 85%"
	case bus.ToolLocation: result.Success = true; result.Data = "Ubicación: Lat 19.4326, Lon -99.1332"
	case bus.ToolDevice: result.Success = true; result.Data = "Nexo Runtime v1.0"
	case bus.ToolWifi: result.Success = true; result.Data = "WiFi: Conectado"
	case bus.ToolClipboardGet: result.Success = true; result.Data = "Portapapeles: vacío"
	case bus.ToolRead: result.Success = true; result.Data = fmt.Sprintf("Leyendo: %s", params["path"])
	case bus.ToolShell: result.Success = true; result.Data = fmt.Sprintf("Ejecutando: %s", params["command"])
	case bus.ToolCamera: result.Success = true; result.Data = "Cámara activada"
	case bus.ToolTorchOn: result.Success = true; result.Data = "Linterna encendida"
	case bus.ToolTorchOff: result.Success = true; result.Data = "Linterna apagada"
	case bus.ToolToast: result.Success = true; result.Data = fmt.Sprintf("Toast: %s", params["message"])
	case bus.ToolTTS: result.Success = true; result.Data = fmt.Sprintf("TTS: %s", params["text"])
	case bus.ToolVibrate: result.Success = true; result.Data = "Vibrando..."
	case bus.ToolNotification: result.Success = true; result.Data = fmt.Sprintf("Notificación: %s", params["title"])
	case bus.ToolIntentView: result.Success = true; result.Data = fmt.Sprintf("Abriendo: %s", params["url"])
	case bus.ToolIntentDial: result.Success = true; result.Data = fmt.Sprintf("Marcando: %s", params["number"])
	case bus.ToolIntentSettings: result.Success = true; result.Data = "Abriendo configuración"
	case bus.ToolIntentAlarm: result.Success = true; result.Data = fmt.Sprintf("Alarma: %s:%s", params["hour"], params["minute"])
	case bus.ToolShizuku: result.Success = true; result.Data = fmt.Sprintf("Shizuku: %s", params["command"])
	default: result.Success = true; result.Data = fmt.Sprintf("Herramienta: %s", tool.Name)
	}
	result.Latency = td.clock.NowMilli() - start
	return result
}

func (td *ToolDecider) emitToolResult(result bus.ToolResult, pkt bus.CognitivePacket) {
	payload, _ := json.Marshal(result)
	td.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("tool_%s", pkt.ID), Type: bus.ToolResult, Source: "tool_decider",
		Target: "control_planner", Priority: 80, Timestamp: td.clock.NowMilli(),
		Payload: payload, Tags: []string{"tool_result", string(result.ToolName)}, TTL: 5,
	})
}

func (td *ToolDecider) GetHistory() []bus.ToolResult {
	td.mu.RLock(); defer td.mu.RUnlock()
	result := make([]bus.ToolResult, len(td.history))
	copy(result, td.history)
	return result
}
