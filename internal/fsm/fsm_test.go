package fsm

import "testing"

func TestFsm_Transit(t *testing.T) {
	state, step := NewFsm()

	newStepId := 3
	id, err := state.Transit(step, newStepId)

	if err != nil {
		t.Error(err.Error())
		return
	}

	if id != newStepId {
		t.Errorf("wrong id: %d -> %d", id, newStepId)
		return
	}

	t.Log(id)
}
