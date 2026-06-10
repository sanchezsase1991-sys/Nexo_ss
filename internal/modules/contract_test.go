package modules

import (
	"testing"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type fakeClock struct{}

func (f *fakeClock) Now() time.Time  { return time.Now() }
func (f *fakeClock) NowMilli() int64 { return time.Now().UnixMilli() }

func TestModuleContract(t *testing.T) {
	var clock scheduler.Clock = &fakeClock{}
	_ = NewSignalTagger(DefaultWeights, clock)
	_ = NewStateRegister(clock)
	_ = NewInputAcquisition(clock)
	_ = NewOutputSink()
}
