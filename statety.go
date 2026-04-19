package statety

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"sync"
)

const (
	Empty Result = iota
	Final
	Stop
)

type (
	CallbackProvider[State comparable, Event comparable] interface {
		Before(ctx context.Context, current State) error
		After(ctx context.Context, event Event, newState State) error
	}

	Converter[State comparable, Event comparable, Payload any] interface {
		CurrentState(ctx context.Context, p Payload) (State, error)
	}

	Setup[State comparable, Event comparable, Payload any] struct {
		StartState  State
		StopStates  []State
		FinalStates []State
		SaveStates  []State

		Config map[State]Steps[State, Event, Payload]

		stop  map[State]struct{}
		final map[State]struct{}
		save  map[State]struct{}
	}

	Steps[State comparable, Event comparable, Payload any] struct {
		Do   func(ctx context.Context, payload Payload) (Event, error)
		Save func(ctx context.Context, payload Payload) error
		Next map[Event]State
	}

	Machine[State comparable, Event comparable, Payload sync.Locker] struct {
		setup            Setup[State, Event, Payload]
		callbackProvider CallbackProvider[State, Event]
		converter        Converter[State, Event, Payload]
	}

	Result int
)

func NewMachine[State comparable, Event comparable, Payload sync.Locker](setup Setup[State, Event, Payload], callbackProvider CallbackProvider[State, Event], converter Converter[State, Event, Payload]) (empty Machine[State, Event, Payload], _ error) {
	setup.stop = maps.Collect(allKeys[State, struct{}](setup.StopStates, struct{}{}))
	setup.final = maps.Collect(allKeys[State, struct{}](setup.FinalStates, struct{}{}))
	setup.save = maps.Collect(allKeys[State, struct{}](setup.SaveStates, struct{}{}))

	for state, route := range setup.Config {
		if _, found := setup.final[state]; !found && route.Do == nil {
			return empty, fmt.Errorf("no do function state: %v", state)
		}
		if _, found := setup.save[state]; found && route.Save == nil {
			return empty, fmt.Errorf("no save function state: %v", state)
		}
	}

	return Machine[State, Event, Payload]{
		setup:            setup,
		callbackProvider: callbackProvider,
		converter:        converter,
	}, nil
}

func allKeys[E comparable, T any](s []E, t T) iter.Seq2[E, T] {
	return func(yield func(E, T) bool) {
		for _, v := range s {
			if !yield(v, t) {
				return
			}
		}
	}
}

func (m *Machine[State, Event, Payload]) Work(ctx context.Context, p Payload) (_ Result, err error) {
	p.Lock()
	defer p.Unlock()

	currentState := m.setup.StartState

	if m.converter != nil {
		currentState, err = m.converter.CurrentState(ctx, p)
		if err != nil {
			return Empty, err
		}
	}

	for {
		if _, ok := m.setup.final[currentState]; ok {
			return Final, nil
		}

		route := m.setup.Config[currentState]

		if _, ok := m.setup.save[currentState]; ok {
			if err = route.Save(ctx, p); err != nil {
				return Empty, err
			}
		}

		if m.callbackProvider != nil {
			if err = m.callbackProvider.Before(ctx, currentState); err != nil {
				return Empty, err
			}
		}

		event, err := route.Do(ctx, p)
		if err != nil {
			return Empty, err
		}

		newState, ok := route.Next[event]
		if !ok {
			return Empty, fmt.Errorf("no transition from state %v on event %v", currentState, event)
		}

		if m.callbackProvider != nil {
			if err = m.callbackProvider.After(ctx, event, newState); err != nil {
				return Empty, err
			}
		}

		if _, ok := m.setup.stop[newState]; ok {
			return Stop, nil
		}

		currentState = newState
	}
}
