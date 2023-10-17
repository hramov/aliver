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
					Id:                 Off,
					Title:              "INSTANCE OFF",
					AllowedTransitions: []int{Undiscovered},
				},
				{
					Id:                 Undiscovered,
					Title:              "UNDISCOVERED",
					AllowedTransitions: []int{Discovered},
				},
				{
					Id:                 Discovered,
					Title:              "DISCOVERED",
					AllowedTransitions: []int{Election, Follower},
				},
				{
					Id:                 Election,
					Title:              "ELECTION",
					AllowedTransitions: []int{PreLeader, Follower},
				},
				{
					Id:                 PreLeader,
					Title:              "PRELEADER",
					AllowedTransitions: []int{Leader, Follower},
				},
				{
					Id:                 Leader,
					Title:              "LEADER",
					AllowedTransitions: []int{Off, Election},
				},
				{
					Id:                 Follower,
					Title:              "FOLLOWER",
					AllowedTransitions: []int{Off, Election},
				},
			},
			Transitions: map[int]Step{},
		}, &Step{
			Id:                 Undiscovered,
			Title:              "UNDISCOVERED",
			AllowedTransitions: []int{Discovered},
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
