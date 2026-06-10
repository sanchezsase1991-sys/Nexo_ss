package main

import (
	"log"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/llm"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/memory"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/modules"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

func main() {
	clock := &scheduler.RealClock{}
	sched := scheduler.NewScheduler(10, clock)
	sched.SetResolver(scheduler.NewDefaultPriorityResolver(scheduler.DefaultRoutingPolicy))
	sched.SetFairnessController(scheduler.NewDefaultFairnessController(100))

	mem, err := memory.NewMemoryBus("./nexo_memory.db", 1024)
	if err != nil {
		log.Fatal(err)
	}
	defer mem.Close()

	// ─── Registro de Estado ───
	state := modules.NewStateRegister(clock)
	state.SetScheduler(sched)
	state.StartDecay()
	defer state.Stop()

	// ─── Intérprete ───
	interpreter := modules.NewDefaultInterpreter()

	// ─── Módulos Base ───
	reframer := modules.NewReframer(state, clock)
	reframer.SetScheduler(sched)

	tagger := modules.NewSignalTagger(modules.DefaultWeights, clock)
	tagger.SetScheduler(sched)
	tagger.SetStateRegister(state)

	wm := modules.NewWorkingMemoryManager(mem, clock)
	wm.SetScheduler(sched)

	predictive := modules.NewPredictiveSimulator(state, modules.DefaultWeights, clock)
	predictive.SetScheduler(sched)

	socialAnalyzer := modules.NewSocialContextAnalyzer(state, clock, mem)
	socialAnalyzer.SetScheduler(sched)

	arr := modules.NewAutoResponseRegulator(state, clock)
	arr.SetScheduler(sched)

	ltm := modules.NewLongTermMemory(state, mem, clock)
	ltm.SetScheduler(sched)

	mcm := modules.NewMetaCognitionMonitor(state, wm, clock)
	mcm.SetScheduler(sched)
	mcm.SetAutoResponseRegulator(arr)
	mcm.SetLongTermMemory(ltm)

	toolDecider := modules.NewToolDecider(state, clock)
	toolDecider.SetScheduler(sched)

	attentionCtrl := modules.NewAttentionController(state, clock)
	attentionCtrl.SetScheduler(sched)

	circuitBreaker := modules.NewCircuitBreaker(100, 30000)

	// ─── LLM ───
	llmPool := llm.NewLLMPool()
	reasoningWorker, err := llm.NewWorker(1, "./model/qwen2.5-coder-3b-Q4_K_M.gguf", 8192, 4, "reasoning")
	if err != nil {
		log.Printf("[NEXO] llama.cpp worker not available: %v — using fallback", err)
	} else {
		llmPool.AddWorker("reasoning", reasoningWorker)
	}
	llmBridge := modules.NewLLMBridge(llmPool, clock)
	llmBridge.SetScheduler(sched)

	// ─── Planificador Central ───
	planner := modules.NewControlPlanner(state, clock, interpreter)
	planner.SetScheduler(sched)
	planner.SetPredictiveSimulator(predictive)
	planner.SetSocialAnalyzer(socialAnalyzer)
	planner.SetWorkingMemory(wm)

	// ─── Salida ───
	formatter := modules.NewOutputFormatter(state, clock, interpreter)
	formatter.SetScheduler(sched)

	input := modules.NewInputAcquisition(clock)
	input.SetScheduler(sched)

	sink := modules.NewOutputSink()

	// ─── 8 NUEVOS MÓDULOS ───
	inhibitoryCtrl := modules.NewInhibitoryControl(state, socialAnalyzer, clock)
	inhibitoryCtrl.SetScheduler(sched)

	resourceEst := modules.NewResourceEstimator(state, clock)
	resourceEst.SetScheduler(sched)

	fatigueComp := modules.NewFatigueCompensator(state, wm, clock)
	fatigueComp.SetScheduler(sched)

	reprUpdater := modules.NewRepresentationUpdater(state, clock)
	reprUpdater.SetScheduler(sched)

	assocManip := modules.NewAssociativeManipulator(wm, ltm, clock)
	assocManip.SetScheduler(sched)

	knowledgeLinker := modules.NewKnowledgeLinker(ltm, clock)
	knowledgeLinker.SetScheduler(sched)

	outputMon := modules.NewOutputMonitor(state, clock)
	outputMon.SetScheduler(sched)

	semanticNet := modules.NewSemanticNetwork(clock)
	semanticNet.SetScheduler(sched)

	// ─── REGISTRO DE HANDLERS ───
	shutdownCh := make(chan struct{})

	sched.Register(bus.Perception, tagger.Handle)

	sched.Register(bus.Thought, reframer.Handle)
	sched.Register(bus.Thought, state.Handle)
	sched.Register(bus.Thought, wm.Handle)
	sched.Register(bus.Thought, predictive.Handle)
	sched.Register(bus.Thought, socialAnalyzer.Handle)
	sched.Register(bus.Thought, toolDecider.Handle)
	sched.Register(bus.Thought, llmBridge.Handle)
	sched.Register(bus.Thought, planner.Handle)

	// 8 nuevos handlers de Thought
	sched.Register(bus.Thought, inhibitoryCtrl.Handle)
	sched.Register(bus.Thought, reprUpdater.Handle)
	sched.Register(bus.Thought, assocManip.Handle)
	sched.Register(bus.Thought, knowledgeLinker.Handle)
	sched.Register(bus.Thought, semanticNet.Handle)

	sched.Register(bus.Meta, state.Handle)
	sched.Register(bus.Meta, ltm.Handle)
	sched.Register(bus.Meta, mcm.Handle)
	sched.Register(bus.Meta, attentionCtrl.Handle)
	sched.Register(bus.Meta, fatigueComp.Handle)

	sched.Register(bus.Memory, wm.Handle)
	sched.Register(bus.Memory, ltm.Handle)

	sched.Register(bus.Action, arr.Handle)
	sched.Register(bus.Action, ltm.Handle)
	sched.Register(bus.Action, formatter.Handle)
	sched.Register(bus.Action, outputMon.Handle)

	sched.Register(bus.Output, sink.Handle)
	sched.Register(bus.Output, outputMon.Handle)

	// ─── LOOPS DE FONDO ───
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C { sched.Tick() }
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C { state.DecayTick() }
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C { wm.DecayChunks(state.GetState()) }
	}()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C { tagger.FlushMediumQueue() }
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C { tagger.FlushLowQueue() }
	}()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C { tagger.PromoteFromLow(state.GetState()) }
	}()

	mcm.StartMonitoring()
	attentionCtrl.StartMonitoring()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C { ltm.PeriodicArchive() }
	}()

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-shutdownCh: return
			case <-ticker.C:
				for _, pkt := range sched.FlushOutput() { sink.Handle(pkt) }
			}
		}
	}()

	log.Println("[NEXO] Sistema completo. 8 nuevos módulos integrados.")

	go func() {
		time.Sleep(1 * time.Second)
		input.ReceiveSignal("Hola sistema", "user", 0.6, []string{"social"})
		time.Sleep(2 * time.Second)
		input.ReceiveSignal("Necesito ayuda urgente", "user", 0.9, []string{"urgent"})
		time.Sleep(2 * time.Second)
		input.ReceiveSignal("¿Qué batería tengo?", "user", 0.5, []string{"question", "tool_request"})
	}()

	_ = circuitBreaker
	_ = resourceEst
	select {}
}
