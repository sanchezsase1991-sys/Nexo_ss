package scheduler

import "time"

type Clock interface {
	Now() time.Time
	NowMilli() int64
}

type RealClock struct{}

func (RealClock) Now() time.Time  { return time.Now() }
func (RealClock) NowMilli() int64 { return time.Now().UnixMilli() }

type MockClock struct{ T int64 }

func (m *MockClock) Now() time.Time  { return time.UnixMilli(m.T) }
func (m *MockClock) NowMilli() int64 { return m.T }
func (m *MockClock) Advance(ms int64) { m.T += ms }
