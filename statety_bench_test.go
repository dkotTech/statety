package statety_test

import (
	"context"
	"testing"

	"github.com/dkotTech/statety"
)

type benchState int
type benchEvent int

const (
	stateA benchState = iota
	stateB
	stateC
	stateDone
)

const (
	evNext benchEvent = iota
	evDone
)

type benchPayload struct{}

func BenchmarkMachineWork(b *testing.B) {
	setup := statety.Setup[benchState, benchEvent, *benchPayload]{
		StartState:  stateA,
		FinalStates: []benchState{stateDone},
		Config: map[benchState]statety.Steps[benchState, benchEvent, *benchPayload]{
			stateA: {
				Do:   func(ctx context.Context, p *benchPayload) (benchEvent, error) { return evNext, nil },
				Next: map[benchEvent]benchState{evNext: stateB},
			},
			stateB: {
				Do:   func(ctx context.Context, p *benchPayload) (benchEvent, error) { return evNext, nil },
				Next: map[benchEvent]benchState{evNext: stateC},
			},
			stateC: {
				Do:   func(ctx context.Context, p *benchPayload) (benchEvent, error) { return evDone, nil },
				Next: map[benchEvent]benchState{evDone: stateDone},
			},
			stateDone: {
				Do:   func(ctx context.Context, p *benchPayload) (benchEvent, error) { return evDone, nil },
				Next: map[benchEvent]benchState{},
			},
		},
	}

	m, err := statety.NewMachine(setup, nil, nil)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()

	for b.Loop() {
		p := &benchPayload{}
		if err := m.Work(ctx, p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMachineWorkParallel(b *testing.B) {
	setup := statety.Setup[benchState, benchEvent, *benchPayload]{
		StartState:  stateA,
		FinalStates: []benchState{stateDone},
		Config: map[benchState]statety.Steps[benchState, benchEvent, *benchPayload]{
			stateA: {
				Do:   func(ctx context.Context, p *benchPayload) (benchEvent, error) { return evNext, nil },
				Next: map[benchEvent]benchState{evNext: stateB},
			},
			stateB: {
				Do:   func(ctx context.Context, p *benchPayload) (benchEvent, error) { return evNext, nil },
				Next: map[benchEvent]benchState{evNext: stateC},
			},
			stateC: {
				Do:   func(ctx context.Context, p *benchPayload) (benchEvent, error) { return evDone, nil },
				Next: map[benchEvent]benchState{evDone: stateDone},
			},
			stateDone: {
				Do:   func(ctx context.Context, p *benchPayload) (benchEvent, error) { return evDone, nil },
				Next: map[benchEvent]benchState{},
			},
		},
	}

	m, err := statety.NewMachine(setup, nil, nil)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p := &benchPayload{}
			if err := m.Work(ctx, p); err != nil {
				b.Fatal(err)
			}
		}
	})
}
