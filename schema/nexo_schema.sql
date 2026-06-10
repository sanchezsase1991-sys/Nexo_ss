-- ============================================================
-- NEXO — Schema completo de SQLite
-- ============================================================

PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA temp_store=MEMORY;
PRAGMA mmap_size=268435456;

-- 1. MEMORIA DE TRABAJO
CREATE TABLE IF NOT EXISTS working_memory (
    id TEXT PRIMARY KEY, payload TEXT, relevance REAL DEFAULT 0.5,
    tier TEXT, connections TEXT, state_footprint REAL DEFAULT 0.0, timestamp INTEGER
);
CREATE INDEX IF NOT EXISTS idx_wm_tier ON working_memory(tier);

-- 2. MEMORIA DE TRABAJO AVANZADA
CREATE TABLE IF NOT EXISTS working_memory_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT, item_id TEXT NOT NULL UNIQUE,
    item_type TEXT NOT NULL, payload TEXT, relevance REAL DEFAULT 0.5,
    connections TEXT, state_footprint REAL DEFAULT 0.0,
    created_at INTEGER, last_accessed_at INTEGER, access_count INTEGER DEFAULT 1,
    evicted BOOLEAN DEFAULT FALSE, eviction_reason TEXT
);
CREATE INDEX IF NOT EXISTS idx_wmi_type ON working_memory_items(item_type, evicted);

-- 3. SEÑALES INHIBITORIAS
CREATE TABLE IF NOT EXISTS inhibitory_signals (
    id INTEGER PRIMARY KEY AUTOINCREMENT, signal_type TEXT NOT NULL,
    source_module TEXT NOT NULL, target_module TEXT NOT NULL,
    priority INTEGER DEFAULT 0, thought_id TEXT, created_at INTEGER,
    processed BOOLEAN DEFAULT FALSE, latency_ms INTEGER,
    cost_saturacion REAL DEFAULT 0.0, cost_intensidad REAL DEFAULT 0.0
);

-- 4. COLA DE SEÑALES
CREATE TABLE IF NOT EXISTS signal_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT, signal_id TEXT NOT NULL,
    payload TEXT, relevance_score REAL DEFAULT 0.0, tier TEXT NOT NULL,
    tags TEXT, intensity REAL DEFAULT 0.5, source TEXT,
    enqueued_at INTEGER, processed BOOLEAN DEFAULT FALSE,
    promoted_from TEXT, promotion_reason TEXT
);

-- 5. MEMORIA EPISÓDICA
CREATE TABLE IF NOT EXISTS episodic_log (
    id TEXT PRIMARY KEY, context TEXT, state_footprint REAL,
    valencia REAL, intensidad REAL, saturacion REAL,
    details TEXT, significance REAL, timestamp INTEGER
);

-- 6. RED SEMÁNTICA
CREATE TABLE IF NOT EXISTS semantic_graph (
    node_id TEXT PRIMARY KEY, node_data TEXT, edges TEXT, strength REAL DEFAULT 0.5
);

-- 7. DIAGNÓSTICOS
CREATE TABLE IF NOT EXISTS diagnostics_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp INTEGER,
    saturacion REAL, intensidad REAL, valencia REAL, stress REAL,
    cognitive_capacity REAL, attention_mode TEXT, overflow BOOLEAN,
    false_positives INTEGER, bugs_detected INTEGER, inhibit_count INTEGER,
    recommendation TEXT
);

-- 8. TRAZAS DE DECISIÓN
CREATE TABLE IF NOT EXISTS decision_traces (
    id INTEGER PRIMARY KEY AUTOINCREMENT, thought_id TEXT,
    urgencia REAL, carga REAL, riesgo REAL, valores REAL,
    score REAL, decision TEXT, timestamp INTEGER
);

-- 9. AGENTES
CREATE TABLE IF NOT EXISTS agent_profiles (
    agent_id TEXT PRIMARY KEY, name TEXT,
    relationship_type TEXT DEFAULT 'stranger', familiarity REAL DEFAULT 0.1,
    trust_score REAL DEFAULT 0.5, emotional_valence REAL DEFAULT 0.0,
    last_interaction INTEGER, interaction_count INTEGER DEFAULT 0,
    communication_style TEXT DEFAULT 'unknown', predicted_state TEXT DEFAULT 'neutral',
    inconsistencies INTEGER DEFAULT 0
);

-- 10. HERRAMIENTAS
CREATE TABLE IF NOT EXISTS tool_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT, tool_name TEXT NOT NULL,
    success BOOLEAN, data TEXT, error TEXT, latency_ms INTEGER, timestamp INTEGER
);

-- 11. RESPUESTAS PRECOMPILADAS
CREATE TABLE IF NOT EXISTS precompiled_responses (
    id INTEGER PRIMARY KEY AUTOINCREMENT, patterns TEXT NOT NULL,
    response TEXT NOT NULL, tone TEXT DEFAULT 'balanced',
    confidence REAL DEFAULT 0.9, usage_count INTEGER DEFAULT 0,
    inhibit_count INTEGER DEFAULT 0
);

-- 12. ATENCIÓN SOSTENIDA
CREATE TABLE IF NOT EXISTS attention_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT, state TEXT NOT NULL,
    purpose_signal REAL, cognitive_capacity REAL,
    focus_duration_seconds REAL, timestamp INTEGER
);

-- 13. CIRCUIT BREAKER
CREATE TABLE IF NOT EXISTS circuit_breaker_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT, event TEXT NOT NULL,
    state TEXT NOT NULL, load_count INTEGER, timestamp INTEGER
);
