package dorm

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Stateful Behavior implments a state machine.
type Stateful struct {
	MachineState *string    `sql:"TYPE:varchar(96);null" json:"machine_state"`
	StatedAt     *time.Time `sql:"TYPE:datetime;null;" json:"stated_at" insert:"no" update:"no"`
}

type StatefulKind struct {
	MachineKind  string     `sql:"TYPE:varchar(96);not null;DEFAULT:'default'" json:"machine_kind"`
	MachineState *string    `sql:"TYPE:varchar(96);null" json:"machine_state"`
	StatedAt     *time.Time `sql:"TYPE:datetime;null;" json:"stated_at" insert:"no" update:"no"`
}

type Stateful4 struct {
	MachineState *string    `sql:"TYPE:varchar(96);null" json:"machine_state"`
	StatedAt     *time.Time `sql:"TYPE:datetime(4);null;" json:"stated_at" insert:"no" update:"no"`
}

type StatefulKind4 struct {
	MachineKind  string     `sql:"TYPE:varchar(96);not null;DEFAULT:'default'" json:"machine_kind"`
	MachineState *string    `sql:"TYPE:varchar(96);null" json:"machine_state"`
	StatedAt     *time.Time `sql:"TYPE:datetime(4);null;" json:"stated_at" insert:"no" update:"no"`
}

// StateMachine is the database table that contains all the states
// and transitions for a given entity that implements Stateful behavior
type StateMachine struct {
	PKey

	// Entity for which this state machine is being defined
	Entity string `sql:"TYPE:varchar(96);not null" json:"entity" insert:"must" update:"no"`

	// To support multiple state machines for an entity
	Kind *string `sql:"TYPE:varchar(96);not null" json:"kind" unique:"uniq_entity_kind(entity,kind)"`

	// All possible states
	States *JArrStr `sql:"TYPE:json" json:"states" insert:"must"`

	// Initial States that the State Machine can begin at
	EntryStates *JArrStr `sql:"TYPE:json" json:"entry_states"`

	// Default initial State (for new records), in case NO state is specified
	DefaultState *string `sql:"TYPE:varchar(96);null" json:"default_state"`

	// State transitions
	Transitions *Movements `sql:"TYPE:json" json:"transitions"`

	SoftDelete
	Historic
	WhosThat
	Timed
}

func (s StateMachine) OnSerialize() error {

	// states should be defined
	if s.States == nil {
		return fmt.Errorf("[states] can't be null")
	}

	// states should not be empty
	if s.States.Contains("") {
		return fmt.Errorf("[states] cannot be empty")
	}

	// transitions must contain states already defined
	if s.Transitions != nil {
		for _, m := range *s.Transitions {
			err := m.validate(*s.States)
			if err != nil {
				return fmt.Errorf("[transitions] %s", err)
			}
		}
	}

	// intial states must be sub-set of overall states
	if s.EntryStates != nil {
		for _, i := range *s.EntryStates {
			if !s.States.Contains(i) {
				return fmt.Errorf("[entry_states]: %s must be one of %s", i, *s.States)
			}
		}
	}

	// default state must be one of entry states (if defined).
	// else default state must be one of overall states.
	if s.DefaultState != nil {
		if !s.States.Contains(*s.DefaultState) {
			return fmt.Errorf("[default_state]: %s must be one of states %s", *s.DefaultState, *s.States)
		}

		if s.EntryStates != nil && !s.EntryStates.Contains(*s.DefaultState) {
			return fmt.Errorf("[default_state]: %s must be one of entry_states %s", *s.DefaultState, *s.EntryStates)
		}
	}

	return nil
}

type Movement struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (t Movement) validate(states JArrStr) error {

	if t.From == "" {
		return errors.New("[from] cannot be empty")
	}

	if !states.Contains(t.From) {
		return fmt.Errorf("[from]: %s must be one of %s", t.From, states)
	}

	if t.To == "" {
		return errors.New("[to] cannot be empty")
	}

	if !states.Contains(t.To) {
		return fmt.Errorf("[to]: %s must be one of %s", t.To, states)
	}

	return nil
}

type Movements []Movement

func NewMovements(items ...Movement) *Movements {
	len := len(items)
	arr := make(Movements, len)
	for i := 0; i < len; i++ {
		arr[i] = items[i]
	}
	return &arr
}

func (j *Movements) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	str, err := json.Marshal(j)
	return string(str), err
}

func (j *Movements) Scan(value interface{}) error {
	if value == nil {
		j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Scan source was not []bytes")
	}
	if err := json.Unmarshal(bytes, &j); err != nil {
		return err
	}
	return nil
}

// StateLog is database table that contains all the
// transitions of state machines of different entities.
// A separate process reads from this table, and dispatches the events
// asynchronously to all the listeners (catchers)
type StateLog struct {
	PKey
	Entity   string  `sql:"TYPE:varchar(64);not null;" json:"entity" insert:"must" update:"no"`
	EntityID uint    `sql:"not null;" json:"entity_id" insert:"must" update:"no"`
	OldState *string `sql:"TYPE:varchar(128)" json:"old_state"`
	NewState *string `sql:"TYPE:varchar(128)" json:"new_state"`
	Timed
	ProcessedAt *time.Time `sql:"null" json:"processed_at" index:"true"`
	Error       *uint8     `sql:"TYPE:tinyint(1) unsigned;null" json:"error"`
	WhosThat
}

type StateLog4 struct {
	PKey
	Entity   string  `sql:"TYPE:varchar(64);not null;" json:"entity" insert:"must" update:"no"`
	EntityID uint    `sql:"not null;" json:"entity_id" insert:"must" update:"no"`
	OldState *string `sql:"TYPE:varchar(128)" json:"old_state"`
	NewState *string `sql:"TYPE:varchar(128)" json:"new_state"`
	Timed4Lite
	ProcessedAt *time.Time `sql:"null" json:"processed_at" index:"true"`
	Error       *uint8     `sql:"TYPE:tinyint(1) unsigned;null" json:"error"`
	WhosThat
}

func (sq StateLog4) TableName() string {
	return "state_log"
}
