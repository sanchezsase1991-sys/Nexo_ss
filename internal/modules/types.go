package modules

import "github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"

type DecisionState int

const (
	DecisionBlocked DecisionState = iota
	DecisionPendingReassessment
	DecisionPartialMatch
	DecisionAligned
)

func (ds DecisionState) String() string {
	switch ds {
	case DecisionBlocked: return "BLOCKED"
	case DecisionPendingReassessment: return "PENDING"
	case DecisionPartialMatch: return "PARTIAL"
	case DecisionAligned: return "ALIGNED"
	default: return "UNKNOWN"
	}
}

type ModuleAlignment struct{ Heuristica, Analisis, Validacion, Trigger bool }
type DecisionInput struct{ UrgenciaExterna, ImportanciaObjetiva, RelevanciaValores, ImpactoSocial, Justicia, Eficiencia, Carga float64 }
type DecisionResult struct{ State DecisionState; Score float64; ShouldExecute, Rumiacion, Reenqueue, Deadlock bool }

type prioridadRelevancia struct{ UrgenciaExterna, ImportanciaObjetiva, RelevanciaValores, ImpactoSocial, Justicia, Eficiencia int }
var DefaultWeights = prioridadRelevancia{30, 20, 95, 90, 85, 25}

type SystemMode int
const (ModeNormal SystemMode = iota; ModeDegraded; ModeCollapse)

type PendingDecision struct{ Packet bus.CognitivePacket; Thought bus.ThoughtState; State SystemState; Timestamp int64 }
type DegradationPhase int
const (PhaseProductiva DegradationPhase = iota; PhaseDegradacion; PhaseAgotamiento; PhaseModoSeguro)

type CapacityImpact struct{ VelocityMod, CalibrationPrecision, ExplicitRecallPenalty float64; SaturationLevel string }
const (SaturationNormal = "normal"; SaturationModerate = "moderate"; SaturationHigh = "high"; SaturationCritical = "critical")

type BufferRetentionEstimate struct{ Seconds float64; Complexity, Description string }
type IntensityThresholds struct{ ActivacionAutomatica, ControlVoluntario, SupresionPosible, SaturacionMemoria, FlowState float64 }
var DefaultIntensityScale = IntensityThresholds{0.6, 0.5, 0.7, 0.5, 0.8}
type DecisionTrace struct{ ThoughtID string; Urgencia, Carga, Riesgo, Valores, Score float64; Decision string }
