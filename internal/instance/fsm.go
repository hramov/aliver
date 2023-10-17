package instance

import (
	"fmt"
	"slices"
)

type Step struct {
	Id                 int
	Title              string
	AllowedTransitions []int
}

type Fsm struct {
	Steps       []Step
	Transitions map[int]Step
}

func NewFsm() (*Fsm, *Step) {
	return &Fsm{
			Steps: []Step{
				{
					Id:                 1,
					Title:              "UNDISCOVERED",
					AllowedTransitions: []int{2, 3},
				},
				{
					Id:                 2,
					Title:              "DISCOVERED",
					AllowedTransitions: []int{3},
				},
				{
					Id:                 3,
					Title:              "ELECTION",
					AllowedTransitions: []int{4, 5},
				},
				{
					Id:                 4,
					Title:              "LEADER",
					AllowedTransitions: []int{1, 2},
				},
				{
					Id:                 5,
					Title:              "FOLLOWER",
					AllowedTransitions: []int{3},
				},
			},
			Transitions: map[int]Step{},
		}, &Step{
			Id:                 1,
			AllowedTransitions: []int{2, 3},
		}
}

func (f *Fsm) Transit(currentStep *Step, newStepId int) error {
	if slices.Contains(currentStep.AllowedTransitions, newStepId) {
		newStep := f.Steps[newStepId-1]
		f.Transitions[currentStep.Id] = newStep
		*currentStep = newStep
		return nil
	}
	return fmt.Errorf("invalid transition: %d -> %d", currentStep.Id, newStepId)
}
