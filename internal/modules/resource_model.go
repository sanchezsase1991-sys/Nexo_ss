package modules

type ResourceModel struct{ CCV, ConsumoConProposito, ConsumoSinProposito, RecuperacionActiva, RecuperacionPasiva float64 }

func (rm *ResourceModel) GetPhase() DegradationPhase {
	p := rm.CCV / 100.0
	switch {
	case p > 0.7: return PhaseProductiva
	case p > 0.4: return PhaseDegradacion
	case p > 0.2: return PhaseAgotamiento
	default: return PhaseModoSeguro
	}
}

func CalculateCCV(base, hwp, hwop, har, hpr float64) float64 {
	return clamp(base-hwp*6.5-hwop*20+har*12.5+hpr*6.5, 0, 100)
}
