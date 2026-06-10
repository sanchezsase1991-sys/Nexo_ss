package bus

type AgentID string

type AgentProfile struct {
	ID                 AgentID  `json:"id"`
	Name               string   `json:"name"`
	RelationshipType   string   `json:"relationship_type"`
	Familiarity        float64  `json:"familiarity"`
	TrustScore         float64  `json:"trust_score"`
	EmotionalValence   float64  `json:"emotional_valence"`
	LastInteraction    int64    `json:"last_interaction"`
	InteractionCount   int      `json:"interaction_count"`
	CommunicationStyle string   `json:"communication_style"`
	PredictedState     string   `json:"predicted_state"`
	Preferences        []string `json:"preferences"`
	Inconsistencies    int      `json:"inconsistencies"`
}

type RelationshipState struct {
	AgentA         AgentID `json:"agent_a"`
	AgentB         AgentID `json:"agent_b"`
	Warmth         float64 `json:"warmth"`
	Respect        float64 `json:"respect"`
	Trust          float64 `json:"trust"`
	Closeness      float64 `json:"closeness"`
	HarmonyScore   float64 `json:"harmony_score"`
	ConflictHistory int    `json:"conflict_history"`
	LastUpdated    int64   `json:"last_updated"`
}

type ValidationResult struct {
	CoherenceScore  float64      `json:"coherence_score"`
	ShouldInhibit   bool         `json:"should_inhibit"`
	InhibitReason   string       `json:"inhibit_reason"`
	OptimalTone     string       `json:"optimal_tone"`
	SocialRisk      float64      `json:"social_risk"`
	ImplicitContent string       `json:"implicit_content"`
	DetectedIntent  string       `json:"detected_intent"`
	AgentContext    AgentProfile `json:"agent_context"`
}

type ContagionResult struct {
	SourceAgent     AgentID            `json:"source_agent"`
	DetectedState   string             `json:"detected_state"`
	DetectedValence float64            `json:"detected_valence"`
	SyncLevel       float64            `json:"sync_level"`
	SuggestedDelta  map[string]float64 `json:"suggested_delta"`
}

type SocialPressureResult struct {
	Pressure        float64 `json:"pressure"`
	Source          string  `json:"source"`
	AgentWaiting    bool    `json:"agent_waiting"`
	ResponseUrgency float64 `json:"response_urgency"`
	ExpectationType string  `json:"expectation_type"`
}
