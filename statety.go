package statety

import (
	"context"
	"errors"
	"fmt"
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
		FinalStates []State

		Config map[State]Steps[State, Event, Payload]

		final map[State]struct{}
	}

	Steps[State comparable, Event comparable, Payload any] struct {
		Do   func(ctx context.Context, payload Payload) (Event, error)
		Save func(ctx context.Context, payload Payload) error
		Next map[Event]State
	}

	Machine[State comparable, Event comparable, Payload any] struct {
		setup            Setup[State, Event, Payload]
		callbackProvider CallbackProvider[State, Event]
		converter        Converter[State, Event, Payload]
	}
)

func NewMachine[State comparable, Event comparable, Payload any](setup Setup[State, Event, Payload], callbackProvider CallbackProvider[State, Event], converter Converter[State, Event, Payload]) (*Machine[State, Event, Payload], error) {
	setup.final = make(map[State]struct{}, len(setup.FinalStates))
	for _, state := range setup.FinalStates {
		setup.final[state] = struct{}{}
	}

	known := func(s State) bool {
		if _, ok := setup.Config[s]; ok {
			return true
		}
		_, ok := setup.final[s]
		return ok
	}

	var errs []error

	if !known(setup.StartState) {
		errs = append(errs, fmt.Errorf("start state %v is not present in Config or FinalStates", setup.StartState))
	}

	for state, route := range setup.Config {
		_, isFinal := setup.final[state]

		switch {
		case isFinal && route.Do != nil:
			errs = append(errs, fmt.Errorf("final state %v must not have Do: it will never run", state))
		case !isFinal && route.Do == nil:
			errs = append(errs, fmt.Errorf("non-final state %v has no Do function", state))
		}

		for event, target := range route.Next {
			if !known(target) {
				errs = append(errs, fmt.Errorf("state %v on event %v transitions to unknown state %v", state, event, target))
			}
		}
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return &Machine[State, Event, Payload]{
		setup:            setup,
		callbackProvider: callbackProvider,
		converter:        converter,
	}, nil
}

func (m *Machine[State, Event, Payload]) Work(ctx context.Context, p Payload) (err error) {
	currentState := m.setup.StartState

	if m.converter != nil {
		currentState, err = m.converter.CurrentState(ctx, p)
		if err != nil {
			return err
		}
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		route, ok := m.setup.Config[currentState]
		if !ok {
			if _, isFinal := m.setup.final[currentState]; isFinal {
				return nil
			}
			return fmt.Errorf("no step for state: %v", currentState)
		}

		if route.Save != nil {
			if err = route.Save(ctx, p); err != nil {
				return err
			}
		}

		if _, ok := m.setup.final[currentState]; ok {
			return nil
		}

		if m.callbackProvider != nil {
			if err = m.callbackProvider.Before(ctx, currentState); err != nil {
				return err
			}
		}

		if route.Do == nil {
			return fmt.Errorf("no do function for state: %v", currentState)
		}

		event, err := route.Do(ctx, p)
		if err != nil {
			return err
		}

		newState, ok := route.Next[event]
		if !ok {
			return fmt.Errorf("no transition from state %v on event %v", currentState, event)
		}

		if m.callbackProvider != nil {
			if err = m.callbackProvider.After(ctx, event, newState); err != nil {
				return err
			}
		}

		currentState = newState
	}
}
