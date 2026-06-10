package scheduler

import "github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"

type PriorityResolver interface {
	Resolve(pkt bus.CognitivePacket) bus.CognitivePacket
	ShouldRequeue(pkt bus.CognitivePacket) (bus.CognitivePacket, bool)
}

type DefaultPriorityResolver struct{ policy RoutingPolicy }

func NewDefaultPriorityResolver(policy RoutingPolicy) *DefaultPriorityResolver {
	return &DefaultPriorityResolver{policy: policy}
}

func (pr *DefaultPriorityResolver) Resolve(pkt bus.CognitivePacket) bus.CognitivePacket {
	pkt.EffectivePriority = pkt.Priority + pr.policy.TypePriority[pkt.Type] + pr.policy.TypePrecedence[pkt.Type]
	return pkt
}

func (pr *DefaultPriorityResolver) ShouldRequeue(pkt bus.CognitivePacket) (bus.CognitivePacket, bool) {
	if pkt.TTL <= 0 {
		return pkt, false
	}
	if pkt.TTL < pkt.InitialTTL/2 && pkt.Priority < 30 {
		return pkt, false
	}
	return pkt, true
}
