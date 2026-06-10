package bus

import (
	"encoding/json"
	"fmt"
)

func (pkt CognitivePacket) AsThoughtState() (ThoughtState, error) {
	var ts ThoughtState
	err := json.Unmarshal(pkt.Payload, &ts)
	if err != nil {
		return ThoughtState{}, fmt.Errorf("AsThoughtState: unmarshal failed for pkt %s: %w", pkt.ID, err)
	}
	if ts.OriginalID == "" {
		ts.OriginalID = pkt.ID
	}
	if ts.Source == "" {
		ts.Source = pkt.Source
	}
	return ts, nil
}
