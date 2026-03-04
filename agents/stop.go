package agents

// Stop cancels the current running task (Run or Stream) if any.
// It is safe to call multiple times; subsequent calls are no-ops.
// This is intended to be called from another goroutine while
// Run or Stream is executing, to asynchronously stop the task.
func (a *Agent) Stop() {
	if a == nil {
		return
	}
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
}
