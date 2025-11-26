package agents

import "time"

// GetDuration returns the total execution duration of the agent's last run.
func (a *Agent) GetDuration() time.Duration {
	return a.Duration
}

// GetStartTime returns the start time of the agent's last run.
func (a *Agent) GetStartTime() time.Time {
	return a.StartTime
}

// GetEndTime returns the end time of the agent's last run.
func (a *Agent) GetEndTime() time.Time {
	return a.EndTime
}

// ResetDuration resets the duration counter, start time, and end time to zero values.
func (a *Agent) ResetDuration() {
	a.Duration = 0
	a.StartTime = time.Time{}
	a.EndTime = time.Time{}
}
