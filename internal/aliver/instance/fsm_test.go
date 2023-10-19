package instance

import "testing"

func TestFsm_Transit(t *testing.T) {
	state, step := NewFsm()

	err := state.Transit(step, Discovered)

	if err != nil {
		t.Error(err.Error())
		return
	}

	if step.Id != Discovered {
		t.Errorf("wrong id: %d -> %d", step.Id, Discovered)
		return
	}

	t.Log(step.Id)
}
