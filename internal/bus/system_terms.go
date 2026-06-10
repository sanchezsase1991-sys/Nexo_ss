package bus

type SystemHierarchy string

const (
	HierarchySystem        SystemHierarchy = "SISTEMA_NEXO"
	HierarchyModule        SystemHierarchy = "MODULO"
	HierarchySubsystem     SystemHierarchy = "SUBSISTEMA"
	HierarchyComponent     SystemHierarchy = "COMPONENTE"
	HierarchyAreaFuncional SystemHierarchy = "AREA_FUNCIONAL"
)

type SystemComponent struct {
	ID               string
	Type             SystemHierarchy
	Parent           string
	Responsabilities []string
}

var NexoArchitecture = map[string]SystemComponent{
	"control_planner":          {ID: "control_planner", Type: HierarchyModule, Parent: "SISTEMA_NEXO", Responsabilities: []string{"orquestación", "decisión", "inhibición"}},
	"meta_cognition_monitor":   {ID: "meta_cognition_monitor", Type: HierarchyModule, Parent: "SISTEMA_NEXO", Responsabilities: []string{"auto-observación", "diagnóstico", "calibración"}},
	"auto_response_regulator":  {ID: "auto_response_regulator", Type: HierarchyModule, Parent: "SISTEMA_NEXO", Responsabilities: []string{"supresión", "respuestas_precompiladas", "disonancia"}},
	"state_register":           {ID: "state_register", Type: HierarchyModule, Parent: "SISTEMA_NEXO", Responsabilities: []string{"valencia", "intensidad", "saturación"}},
	"long_term_memory":         {ID: "long_term_memory", Type: HierarchySubsystem, Parent: "SISTEMA_NEXO", Responsabilities: []string{"archivo_episódico", "red_semántica"}},
	"working_memory":           {ID: "working_memory", Type: HierarchySubsystem, Parent: "SISTEMA_NEXO", Responsabilities: []string{"chunks_temporales", "conexiones_laterales"}},
	"signal_processing":        {ID: "signal_processing", Type: HierarchySubsystem, Parent: "SISTEMA_NEXO", Responsabilities: []string{"etiquetado", "filtrado", "encolamiento"}},
	"tool_decider":             {ID: "tool_decider", Type: HierarchyModule, Parent: "SISTEMA_NEXO", Responsabilities: []string{"ejecución_herramientas", "mcp_bridge"}},
	"llm_bridge":               {ID: "llm_bridge", Type: HierarchyModule, Parent: "SISTEMA_NEXO", Responsabilities: []string{"razonamiento_llm", "nlp"}},
	"attention_controller":     {ID: "attention_controller", Type: HierarchyModule, Parent: "SISTEMA_NEXO", Responsabilities: []string{"atención_sostenida", "detección_fatiga"}},
	"circuit_breaker":          {ID: "circuit_breaker", Type: HierarchyModule, Parent: "SISTEMA_NEXO", Responsabilities: []string{"protección_carga", "degradación_controlada"}},
}
