package bus

type Tier string

const (
	TierHigh   Tier = "ALTA"
	TierMedium Tier = "MEDIA"
	TierLow    Tier = "BAJA"
)

type ThoughtState struct {
	OriginalID string   `json:"original_id"`
	Tags       []string `json:"tags"`
	Tier       Tier     `json:"tier"`
	Payload    string   `json:"payload"`
	Intensity  float64  `json:"intensity"`
	Score      float64  `json:"relevance_score"`
	Source     string   `json:"source"`
}

func (t ThoughtState) IsUrgent() bool            { return containsTag(t.Tags, "urgent") }
func (t ThoughtState) IsQuestion() bool          { return containsTag(t.Tags, "question") }
func (t ThoughtState) IsSocial() bool            { return containsTag(t.Tags, "social") }
func (t ThoughtState) IsToolRequest() bool        { return containsTag(t.Tags, "tool_request") }
func (t ThoughtState) HasRichAssociations() bool  { return len(t.Tags) > 2 || containsTag(t.Tags, "emotional") }
func (t ThoughtState) IsFromKnownAgent() bool     { return t.Source != "" && t.Source != "unknown" && t.Source != "user" }
func (t ThoughtState) HasImplicitContent() bool   { return containsTag(t.Tags, "question") || containsTag(t.Tags, "social") }
func (t ThoughtState) IsTemporalDeadline() bool   { return containsTag(t.Tags, "deadline") || containsTag(t.Tags, "urgent") }
func (t ThoughtState) MatchesRecentPattern() bool { return containsTag(t.Tags, "recurring") || containsTag(t.Tags, "pattern_match") }

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
