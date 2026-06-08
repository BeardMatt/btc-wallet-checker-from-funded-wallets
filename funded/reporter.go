package funded

import "time"

// Reporter receives progress and status callbacks during funded data operations.
// Implementations may embed NopReporter and override selected methods.
type Reporter interface {
	Section(title string)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	PrintfErr(format string, args ...any)
	PrintlnErr(args ...any)
	Progress(label string, current, total int64)
	ProgressIndeterminate(label string)
	ClearProgress()
	LogFundedLoad(sets Sets, elapsed time.Duration, fromCache bool)
	LogBloomBuild(sets Sets, elapsed time.Duration)
	Quiet() bool
	Verbose() bool
}

// NopReporter is a no-op Reporter suitable as a default or embed base.
type NopReporter struct{}

func (NopReporter) Section(string)                                              {}
func (NopReporter) Infof(string, ...any)                                        {}
func (NopReporter) Warnf(string, ...any)                                        {}
func (NopReporter) PrintfErr(string, ...any)                                    {}
func (NopReporter) PrintlnErr(...any)                                           {}
func (NopReporter) Progress(string, int64, int64)                               {}
func (NopReporter) ProgressIndeterminate(string)                                {}
func (NopReporter) ClearProgress()                                              {}
func (NopReporter) LogFundedLoad(Sets, time.Duration, bool)                     {}
func (NopReporter) LogBloomBuild(Sets, time.Duration)                           {}
func (NopReporter) Quiet() bool                                                 { return true }
func (NopReporter) Verbose() bool                                               { return false }