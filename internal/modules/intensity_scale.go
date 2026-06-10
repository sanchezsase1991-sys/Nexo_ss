package modules

func TiempoRetencionBuffer(complejidad float64) BufferRetentionEstimate {
	switch {
	case complejidad > 0.8: return BufferRetentionEstimate{17.5, "alta", "5-30 segundos"}
	case complejidad > 0.5: return BufferRetentionEstimate{3.0, "media", "1-5 segundos"}
	default: return BufferRetentionEstimate{1.75, "baja", "0.5-3 segundos"}
	}
}
