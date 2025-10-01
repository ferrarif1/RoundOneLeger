package models

type storeSnapshot struct {
	entries map[LedgerType][]LedgerEntry
}

type historyStack struct {
	states []storeSnapshot
	future []storeSnapshot
	limit  int
}

func (h *historyStack) Reset(snapshot storeSnapshot) {
	h.states = []storeSnapshot{snapshot}
	h.future = nil
}

func (h *historyStack) Push(snapshot storeSnapshot) {
	h.states = append(h.states, snapshot)
	if h.limit > 0 && len(h.states) > h.limit {
		h.states = h.states[1:]
	}
	h.future = nil
}

func (h *historyStack) Current() storeSnapshot {
	if len(h.states) == 0 {
		return storeSnapshot{}
	}
	return h.states[len(h.states)-1]
}

func (h *historyStack) Undo() (storeSnapshot, error) {
	if len(h.states) <= 1 {
		return storeSnapshot{}, ErrUndoUnavailable
	}
	current := h.states[len(h.states)-1]
	h.states = h.states[:len(h.states)-1]
	h.future = append([]storeSnapshot{current}, h.future...)
	return h.states[len(h.states)-1], nil
}

func (h *historyStack) Redo() (storeSnapshot, error) {
	if len(h.future) == 0 {
		return storeSnapshot{}, ErrRedoUnavailable
	}
	next := h.future[0]
	h.future = h.future[1:]
	h.states = append(h.states, next)
	if h.limit > 0 && len(h.states) > h.limit {
		h.states = h.states[1:]
	}
	return next, nil
}

func (h *historyStack) CanUndo() bool {
	return len(h.states) > 1
}

func (h *historyStack) CanRedo() bool {
	return len(h.future) > 0
}
