package modules

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// EpisynapticMemory — SECCIÓN 11: Memoria Episynaptica de Largo Plazo
// Responsable de:
// 1. Consolidación de memoria episódica
// 2. Vinculación semántica
// 3. Traces de cadena episynaptica
// 4. Aprendizaje de redes sociocognitivas
// 5. Recuperación de memoria a largo plazo

type EpisynapticMemory struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	wm          *WorkingMemoryManager
	mu          sync.RWMutex
	episodes    []EpisodicTrace
	synapses    []SynapticLink
	networks    []SociocognitiveNetwork
	maxEpisodes int
	maxSynapses int
	maxNetworks int
	stats       EpisynapticStats
}

type EpisodicTrace struct {
	ID           string
	Content      string
	State        SystemState
	Tags         []string
	Importance   float64
	Associations []string
	CreatedAt    int64
	LastAccess   int64
	AccessCount  int
	DecayRate    float64
}

type SynapticLink struct {
	ID          string
	SourceID    string
	TargetID    string
	Strength    float64
	Type        SynapticType
	LastFired   int64
	FireCount   int
}

type SynapticType int

const (
	SynapticAssociation SynapticType = iota
	SynapticCausal
	SynapticTemporal
	SynapticSemantic
)

func (st SynapticType) String() string {
	switch st {
	case SynapticAssociation:
		return "ASSOCIATION"
	case SynapticCausal:
		return "CAUSAL"
	case SynapticTemporal:
		return "TEMPORAL"
	case SynapticSemantic:
		return "SEMANTIC"
	default:
		return "UNKNOWN"
	}
}

type SociocognitiveNetwork struct {
	ID        string
	Name      string
	Nodes     []string
	Edges     []SynapticLink
	Strength  float64
	LastUsed  int64
}

type EpisynapticStats struct {
	TotalEpisodes  int
	TotalSynapses  int
	TotalNetworks  int
	AvgImportance  float64
	AvgStrength    float64
	Consolidations int
}

func NewEpisynapticMemory(stateReg *StateRegister, clock scheduler.Clock, wm *WorkingMemoryManager) *EpisynapticMemory {
	return &EpisynapticMemory{
		stateReg:    stateReg,
		clock:       clock,
		wm:          wm,
		episodes:    make([]EpisodicTrace, 0, 256),
		synapses:    make([]SynapticLink, 0, 512),
		networks:    make([]SociocognitiveNetwork, 0, 64),
		maxEpisodes: 256,
		maxSynapses: 512,
		maxNetworks: 64,
	}
}

func (em *EpisynapticMemory) SetScheduler(s *scheduler.Scheduler) { em.sched = s }

func (em *EpisynapticMemory) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Memory:
		em.processMemoryEvent(pkt)
	case bus.Meta:
		em.processMeta(pkt)
	case bus.Thought:
		em.processThoughtForMemory(pkt)
	}
}

func (em *EpisynapticMemory) processMemoryEvent(pkt bus.CognitivePacket) {
	var event map[string]any
	json.Unmarshal(pkt.Payload, &event)

	// Procesar según tipo de evento
	eventType, _ := event["event"].(string)

	switch eventType {
	case "consolidate":
		em.consolidateMemory(event)
	case "recall":
		em.recallMemory(event, pkt)
	case "store":
		em.storeEpisode(event)
	case "link":
		em.createSynapticLink(event)
	case "activate":
		em.activateNetwork(event)
	}
}

func (em *EpisynapticMemory) processMeta(pkt bus.CognitivePacket) {
	var meta map[string]any
	json.Unmarshal(pkt.Payload, &meta)

	// Obtener episodios
	if getEpisodes, ok := meta["get_episodes"].(bool); ok && getEpisodes {
		em.emitEpisodes()
	}

	// Obtener sinapsis
	if getSynapses, ok := meta["get_synapses"].(bool); ok && getSynapses {
		em.emitSynapses()
	}

	// Obtener redes
	if getNetworks, ok := meta["get_networks"].(bool); ok && getNetworks {
		em.emitNetworks()
	}

	// Buscar por tags
	if searchTags, ok := meta["search_tags"].([]string); ok {
		em.searchByTags(searchTags)
	}

	// Consolidar memoria
	if consolidate, ok := meta["consolidate"].(bool); ok && consolidate {
		em.runConsolidation()
	}

	// Estadísticas
	if statsReq, ok := meta["request_stats"].(bool); ok && statsReq {
		stats := em.GetStats()
		payload, _ := json.Marshal(stats)
		em.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("epi_stats_%d", em.clock.NowMilli()), Type: bus.Meta,
			Source: "episynaptic_memory", Target: "meta_cognition",
			Priority: 40, Timestamp: em.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

func (em *EpisynapticMemory) processThoughtForMemory(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil {
		return
	}

	// Evaluar si el pensamiento debe ser almacenado
	if thought.Score > 0.6 || thought.Intensity > 0.7 {
		episode := em.createEpisodeFromThought(thought)
		em.storeEpisodeFromEpisode(episode)
	}

	// Buscar episodios relacionados
	related := em.findRelatedEpisodes(thought)
	if len(related) > 0 {
		em.emitRelatedEpisodes(related, thought)
	}
}

func (em *EpisynapticMemory) consolidateMemory(event map[string]any) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Consolidar episodios débiles en fuertes
	for i := range em.episodes {
		if em.episodes[i].AccessCount > 3 && em.episodes[i].Importance > 0.5 {
			// Aumentar importancia
			em.episodes[i].Importance = clamp(em.episodes[i].Importance+0.1, 0, 1)
			em.episodes[i].DecayRate *= 0.9 // Reducir tasa de decaimiento
		}
	}

	// Consolidar sinapsis débiles
	for i := range em.synapses {
		if em.synapses[i].FireCount > 5 {
			em.synapses[i].Strength = clamp(em.synapses[i].Strength+0.05, 0, 1)
		}
	}

	em.stats.Consolidations++
}

func (em *EpisynapticMemory) recallMemory(event map[string]any, pkt bus.CognitivePacket) {
	query, _ := event["query"].(string)

	// Buscar episodios relevantes
	var relevant []EpisodicTrace
	em.mu.RLock()
	for _, episode := range em.episodes {
		if em.isRelevant(episode, query) {
			relevant = append(relevant, episode)
		}
	}
	em.mu.RUnlock()

	// Emitir resultados
	if len(relevant) > 0 {
		em.emitRecallResults(relevant, pkt)
	}
}

func (em *EpisynapticMemory) storeEpisode(event map[string]any) {
	episode := EpisodicTrace{
		ID:           fmt.Sprintf("ep_%d", em.clock.NowMilli()),
		Tags:         make([]string, 0),
		Associations: make([]string, 0),
		CreatedAt:    em.clock.NowMilli(),
		LastAccess:   em.clock.NowMilli(),
		DecayRate:    0.95,
	}

	if content, ok := event["content"].(string); ok {
		episode.Content = content
	}
	if importance, ok := event["importance"].(float64); ok {
		episode.Importance = clamp(importance, 0, 1)
	}
	if tags, ok := event["tags"].([]string); ok {
		episode.Tags = tags
	}

	em.storeEpisodeFromEpisode(episode)
}

func (em *EpisynapticMemory) createSynapticLink(event map[string]any) {
	link := SynapticLink{
		ID: fmt.Sprintf("syn_%d", em.clock.NowMilli()),
	}

	if source, ok := event["source"].(string); ok {
		link.SourceID = source
	}
	if target, ok := event["target"].(string); ok {
		link.TargetID = target
	}
	if strength, ok := event["strength"].(float64); ok {
		link.Strength = clamp(strength, 0, 1)
	}
	if synType, ok := event["type"].(string); ok {
		switch synType {
		case "association":
			link.Type = SynapticAssociation
		case "causal":
			link.Type = SynapticCausal
		case "temporal":
			link.Type = SynapticTemporal
		case "semantic":
			link.Type = SynapticSemantic
		}
	}

	em.mu.Lock()
	em.synapses = append(em.synapses, link)
	if len(em.synapses) > em.maxSynapses {
		em.synapses = em.synapses[1:]
	}
	em.mu.Unlock()
}

func (em *EpisynapticMemory) activateNetwork(event map[string]any) {
	networkID, _ := event["network_id"].(string)

	em.mu.Lock()
	defer em.mu.Unlock()

	for i := range em.networks {
		if em.networks[i].ID == networkID {
			em.networks[i].LastUsed = em.clock.NowMilli()
			em.networks[i].Strength = clamp(em.networks[i].Strength+0.05, 0, 1)
			break
		}
	}
}

func (em *EpisynapticMemory) createEpisodeFromThought(thought bus.ThoughtState) EpisodicTrace {
	state := em.stateReg.GetState()

	return EpisodicTrace{
		ID:           fmt.Sprintf("ep_%s", thought.OriginalID),
		Content:      thought.Payload,
		State:        state,
		Tags:         thought.Tags,
		Importance:   thought.Score * 0.5 + thought.Intensity * 0.5,
		Associations: make([]string, 0),
		CreatedAt:    em.clock.NowMilli(),
		LastAccess:   em.clock.NowMilli(),
		DecayRate:    0.95,
	}
}

func (em *EpisynapticMemory) storeEpisodeFromEpisode(episode EpisodicTrace) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.episodes = append(em.episodes, episode)
	if len(em.episodes) > em.maxEpisodes {
		em.episodes = em.episodes[1:]
	}
}

func (em *EpisynapticMemory) findRelatedEpisodes(thought bus.ThoughtState) []EpisodicTrace {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var related []EpisodicTrace
	for _, episode := range em.episodes {
		if em.isEpisodeRelated(episode, thought) {
			related = append(related, episode)
		}
	}
	return related
}

func (em *EpisynapticMemory) isRelevant(episode EpisodicTrace, query string) bool {
	// Verificar si el episodio es relevante para la consulta
	for _, tag := range episode.Tags {
		if tag == query {
			return true
		}
	}

	// Verificar contenido
	if len(episode.Content) > 0 && len(query) > 0 {
		// Búsqueda simple de substring
		for i := 0; i <= len(episode.Content)-len(query); i++ {
			if episode.Content[i:i+len(query)] == query {
				return true
			}
		}
	}

	return false
}

func (em *EpisynapticMemory) isEpisodeRelated(episode EpisodicTrace, thought bus.ThoughtState) bool {
	// Verificar tags compartidos
	for _, tag1 := range episode.Tags {
		for _, tag2 := range thought.Tags {
			if tag1 == tag2 {
				return true
			}
		}
	}

	// Verificar similitud de contenido
	if len(episode.Content) > 0 && len(thought.Payload) > 0 {
		// Similitud básica de palabras
		words1 := make(map[string]bool)
		words2 := make(map[string]bool)

		for _, w := range splitWords(episode.Content) {
			words1[w] = true
		}
		for _, w := range splitWords(thought.Payload) {
			words2[w] = true
		}

		common := 0
		for w := range words1 {
			if words2[w] {
				common++
			}
		}

		if common > 2 {
			return true
		}
	}

	return false
}

func (em *EpisynapticMemory) searchByTags(tags []string) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var found []EpisodicTrace
	for _, episode := range em.episodes {
		for _, searchTag := range tags {
			for _, episodeTag := range episode.Tags {
				if searchTag == episodeTag {
					found = append(found, episode)
					break
				}
			}
		}
	}

	if len(found) > 0 {
		payload, _ := json.Marshal(found)
		em.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("tag_search_%d", em.clock.NowMilli()), Type: bus.Memory,
			Source: "episynaptic_memory", Target: "control_planner",
			Priority: 60, Timestamp: em.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

func (em *EpisynapticMemory) runConsolidation() {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Almacenar episodios fuertes en red semántica
	for _, episode := range em.episodes {
		if episode.Importance > 0.7 && episode.AccessCount > 2 {
			// Crear nodo en red semántica
			em.createSemanticNode(episode)
		}
	}

	// Reforzar sinapsis frecuentes
	for i := range em.synapses {
		if em.synapses[i].FireCount > 3 {
			em.synapses[i].Strength = clamp(em.synapses[i].Strength+0.03, 0, 1)
		}
	}
}

func (em *EpisynapticMemory) createSemanticNode(episode EpisodicTrace) {
	// Buscar red existente o crear nueva
	var targetNetwork *SociocognitiveNetwork

	for i := range em.networks {
		if em.networks[i].Strength > 0.5 {
			targetNetwork = &em.networks[i]
			break
		}
	}

	if targetNetwork == nil {
		// Crear nueva red
		network := SociocognitiveNetwork{
			ID:       fmt.Sprintf("net_%d", em.clock.NowMilli()),
			Name:     episode.Tags[0],
			Nodes:    []string{episode.ID},
			Edges:    make([]SynapticLink, 0),
			Strength: 0.3,
			LastUsed: em.clock.NowMilli(),
		}
		em.networks = append(em.networks, network)
	} else {
		// Agregar nodo a red existente
		targetNetwork.Nodes = append(targetNetwork.Nodes, episode.ID)
		targetNetwork.LastUsed = em.clock.NowMilli()
	}
}

func (em *EpisynapticMemory) emitEpisodes() {
	em.mu.RLock()
	defer em.mu.RUnlock()

	payload, _ := json.Marshal(em.episodes)
	em.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("episodes_%d", em.clock.NowMilli()), Type: bus.Memory,
		Source: "episynaptic_memory", Target: "meta_cognition",
		Priority: 40, Timestamp: em.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (em *EpisynapticMemory) emitSynapses() {
	em.mu.RLock()
	defer em.mu.RUnlock()

	payload, _ := json.Marshal(em.synapses)
	em.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("synapses_%d", em.clock.NowMilli()), Type: bus.Memory,
		Source: "episynaptic_memory", Target: "meta_cognition",
		Priority: 40, Timestamp: em.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (em *EpisynapticMemory) emitNetworks() {
	em.mu.RLock()
	defer em.mu.RUnlock()

	payload, _ := json.Marshal(em.networks)
	em.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("networks_%d", em.clock.NowMilli()), Type: bus.Memory,
		Source: "episynaptic_memory", Target: "meta_cognition",
		Priority: 40, Timestamp: em.clock.NowMilli(),
		Payload: payload, TTL: 2,
	})
}

func (em *EpisynapticMemory) emitRecallResults(episodes []EpisodicTrace, pkt bus.CognitivePacket) {
	payload, _ := json.Marshal(episodes)
	em.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("recall_%s", pkt.ID), Type: bus.Memory,
		Source: "episynaptic_memory", Target: pkt.Source,
		Priority: 60, Timestamp: em.clock.NowMilli(),
		Payload: payload, TTL: 3,
	})
}

func (em *EpisynapticMemory) emitRelatedEpisodes(episodes []EpisodicTrace, thought bus.ThoughtState) {
	payload, _ := json.Marshal(map[string]any{
		"episodes": episodes,
		"query":    thought.Payload,
	})

	em.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("related_%s", thought.OriginalID), Type: bus.Memory,
		Source: "episynaptic_memory", Target: "working_memory",
		Priority: 60, Timestamp: em.clock.NowMilli(),
		Payload: payload, TTL: 3,
	})
}

func (em *EpisynapticMemory) GetStats() EpisynapticStats {
	em.mu.Lock()
	defer em.mu.Unlock()

	stats := em.stats
	stats.TotalEpisodes = len(em.episodes)
	stats.TotalSynapses = len(em.synapses)
	stats.TotalNetworks = len(em.networks)

	if stats.TotalEpisodes > 0 {
		totalImportance := 0.0
		for _, ep := range em.episodes {
			totalImportance += ep.Importance
		}
		stats.AvgImportance = totalImportance / float64(stats.TotalEpisodes)
	}

	if stats.TotalSynapses > 0 {
		totalStrength := 0.0
		for _, syn := range em.synapses {
			totalStrength += syn.Strength
		}
		stats.AvgStrength = totalStrength / float64(stats.TotalSynapses)
	}

	return stats
}

func splitWords(s string) []string {
	words := make([]string, 0)
	current := ""
	for _, c := range s {
		if c == ' ' || c == '\n' || c == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}
