package bus

type PacketType string

const (
	Perception  PacketType = "perception"
	Thought     PacketType = "thought"
	Memory      PacketType = "memory"
	Action      PacketType = "action"
	Output      PacketType = "output"
	Meta        PacketType = "meta"
	ToolRequest PacketType = "tool_request"
	ToolResult  PacketType = "tool_result"
)

type CognitivePacket struct {
	ID                string
	CorrelationID     string
	ParentID          string
	Type              PacketType
	Priority          int
	EffectivePriority int
	EnqueuedAt        int64
	FairnessBoost     int
	Source            string
	Target            string
	Timestamp         int64
	Payload           []byte
	Tags              []string
	TTL               int
	InitialTTL        int
	Attempt           int
}
