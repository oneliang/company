package workflow

// Workflow represents a DAG of steps.
type Workflow struct {
	ID     string  `json:"id"`
	Steps  []*Step `json:"steps"`
	Status string  `json:"status"`
}

// GetReadySteps returns steps ready to execute (dependencies satisfied).
func (w *Workflow) GetReadySteps() []*Step {
	var ready []*Step
	for _, step := range w.Steps {
		if step.Status != StepPending {
			continue
		}
		if w.dependenciesMet(step) {
			ready = append(ready, step)
		}
	}
	return ready
}

// dependenciesMet checks if all dependencies are completed.
func (w *Workflow) dependenciesMet(step *Step) bool {
	for _, depID := range step.DependsOn {
		dep := w.GetStep(depID)
		if dep == nil || dep.Status != StepCompleted {
			return false
		}
	}
	return true
}

// GetStep retrieves a step by ID.
func (w *Workflow) GetStep(id string) *Step {
	for _, s := range w.Steps {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// CompleteStep marks a step as completed.
func (w *Workflow) CompleteStep(id string, output string) {
	step := w.GetStep(id)
	if step != nil {
		step.Status = StepCompleted
		step.Output = output
	}
}

// ClearDownstreamSteps recursively clears all steps that depend on the given step.
// Returns list of cleared step IDs.
func (w *Workflow) ClearDownstreamSteps(stepID string) []string {
	var cleared []string
	w.clearDownstreamRecursive(stepID, &cleared)
	return cleared
}

func (w *Workflow) clearDownstreamRecursive(stepID string, cleared *[]string) {
	for _, s := range w.Steps {
		if containsStr(s.DependsOn, stepID) && s.Status != StepPending {
			s.Status = StepPending
			s.Output = ""
			*cleared = append(*cleared, s.ID)
			// Recursively clear downstream steps
			w.clearDownstreamRecursive(s.ID, cleared)
		}
	}
}

// containsStr checks if a string slice contains a string.
func containsStr(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}