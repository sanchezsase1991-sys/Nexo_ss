package bus

type DecisionTrace struct {
	ThoughtID string  `json:"thought_id"`
	Urgencia  float64 `json:"urgencia"`
	Carga     float64 `json:"carga"`
	Riesgo    float64 `json:"riesgo"`
	Valores   float64 `json:"valores"`
	Score     float64 `json:"score"`
	Decision  string  `json:"decision"`
}
