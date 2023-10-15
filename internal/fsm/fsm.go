package fsm

import (
	"fmt"
	"slices"
)

type Step struct {
	Id                 int
	AllowedTransitions []int
}

type Fsm struct {
	Steps       []Step
	Transitions map[int]Step
}

func NewFsm() (*Fsm, *Step) {
	return &Fsm{
			Steps: []Step{
				// undiscovered instance
				{
					Id:                 1,
					AllowedTransitions: []int{2, 3},
				},
				// discovered instance
				{
					Id:                 2,
					AllowedTransitions: []int{3},
				},
				// discovered instance while election
				{
					Id:                 3,
					AllowedTransitions: []int{4, 5},
				},
				// leader
				{
					Id:                 4,
					AllowedTransitions: []int{1, 2},
				},
				// follower
				{
					Id:                 5,
					AllowedTransitions: []int{3},
				},
			},
			Transitions: map[int]Step{},
		}, &Step{
			Id:                 1,
			AllowedTransitions: []int{2, 3},
		}
}

func (f *Fsm) Transit(currentStep *Step, newStepId int) (int, error) {
	if slices.Contains(currentStep.AllowedTransitions, newStepId) {
		newStep := f.Steps[newStepId-1]
		f.Transitions[currentStep.Id] = newStep
		return newStep.Id, nil
	}
	return 0, fmt.Errorf("invalid transition: %d -> %d", currentStep.Id, newStepId)
}
