package modules

import (
	"testing"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
)

func TestModuleContract(t *testing.T) {
	var _ bus.Module = (*SignalTagger)(nil)
	var _ bus.Module = (*ControlPlanner)(nil)
	var _ bus.Module = (*WorkingMemoryManager)(nil)
	var _ bus.Module = (*StateRegister)(nil)
	var _ bus.Module = (*OutputFormatter)(nil)
	var _ bus.Module = (*InputAcquisition)(nil)
	var _ bus.Module = (*OutputSink)(nil)
	var _ bus.Module = (*PredictiveSimulator)(nil)
	var _ bus.Module = (*Reframer)(nil)
	var _ bus.Module = (*SocialContextAnalyzer)(nil)
	var _ bus.Module = (*ToolDecider)(nil)
	var _ bus.Module = (*AttentionController)(nil)
}
