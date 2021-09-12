package dorm

import "github.com/rightjoin/fig"

// If watchers are started on different boxes, the system
// should halt or panic

type Event struct {
	Entity   string `json:"entity"`
	EntityID uint   `json:"entity_id"`
}

type StateEvent struct {
	Event
	Timed
	OldState string `json:"old_state"`
	NewState string `json:"new_state"`
}

func WatchStateUpdate(code string, fn func(StateEvent) bool, models ...interface{}) {

	// If state messages are not being read and pushed
	// to bus, then enable it
	broadcastStateEvents()

}

var broadcastState = false

func broadcastStateEvents() {

	// Run it only once in a process
	if broadcastState {
		return
	}

	// If state events are disabled, then halt/panic
	if !fig.BoolOr(true, "dorm.state-event.active") {
		return
	}

	// Initiate process to read from state queue and
	// send messages to MessageBus
	go pushStateTableToReceivers()

	// TODO: bypass generation of a unique code

	broadcastState = true
}

// pushStateTableToReceivers function should be run once
// during a process. It ensures this be acquiring a lock in the
// database. Then it runs continuously and reads from state_queue
// table and dispatches the rows read to messageBus in the form of
// StateEvent.
func pushStateTableToReceivers() {

	// If state lock is acquired by another process,
	// then panic/halt

	// If state lock is already acquired
	// by this process, then do nothing

	// Aquire the lock: to ensure that this function is
	// called only once

	// Add a callback, so that when this process quits
	// then the lock is released

	// loop
	for {
		if processExiting {
			// clenaup state lock

			// push to channel
		} else {

		}
	}
}
