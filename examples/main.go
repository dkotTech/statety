package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dkotTech/statety"
)

type (
	State string
	Event string

	Order struct {
		ID          int
		UserEmail   string
		Amount      float64
		Attempts    int
		TrackingNum string
		sync.Mutex
	}
)

const (
	StatePending    State = "pending"
	StateProcessing State = "processing"
	StateShipping   State = "shipping"
	StateDelivered  State = "delivered" // финальное
	StateCancelled  State = "cancelled" // стоп

	EventPaid      Event = "paid"
	EventDeclined  Event = "declined"
	EventRetry     Event = "retry"
	EventGiveUp    Event = "give_up"
	EventShipped   Event = "shipped"
	EventDelivered Event = "delivered"
	EventCancelled Event = "cancelled"
)

// --- конфиг машины ---

var config = statety.Setup[State, Event, *Order]{
	StartState:  StatePending,
	FinalStates: []State{StateDelivered, StateCancelled},
	StopStates:  []State{StateShipping},
	SaveStates:  []State{StateShipping, StateDelivered, StateCancelled},

	Config: map[State]statety.Steps[State, Event, *Order]{
		StatePending: {
			Do: func(ctx context.Context, o *Order) (Event, error) {
				fmt.Printf("  [pending] оплата заказа #%d на %.2f руб...\n", o.ID, o.Amount)
				if o.Amount <= 0 {
					return EventCancelled, nil
				}
				return EventPaid, nil
			},
			Next: map[Event]State{
				EventPaid:      StateProcessing,
				EventCancelled: StateCancelled,
			},
		},

		StateProcessing: {
			Do: func(ctx context.Context, o *Order) (Event, error) {
				o.Attempts++
				fmt.Printf("  [processing] попытка оплаты #%d (попытка %d)...\n", o.ID, o.Attempts)

				// симуляция: первые 2 попытки — отказ
				if o.Attempts < 3 {
					return EventDeclined, nil
				}
				o.TrackingNum = fmt.Sprintf("TRACK-%d", o.ID*100)
				return EventShipped, nil
			},
			Save: func(ctx context.Context, payload *Order) error {
				fmt.Printf("[save]")
				return nil
			},
			Next: map[Event]State{
				EventShipped:  StateShipping,
				EventDeclined: StateProcessing, // retry-петля
				EventGiveUp:   StateCancelled,
			},
		},

		StateShipping: {
			Do: func(ctx context.Context, o *Order) (Event, error) {
				fmt.Printf("  [shipping] трек-номер %s, доставляем...\n", o.TrackingNum)
				if o.TrackingNum == "" {
					return EventCancelled, errors.New("трек-номер не назначен")
				}
				return EventDelivered, nil
			},
			Next: map[Event]State{
				EventDelivered: StateDelivered,
				EventCancelled: StateProcessing,
			},
			Save: func(ctx context.Context, payload *Order) error {
				fmt.Printf("[save]")
				return nil
			},
		},

		StateDelivered: {
			Save: func(ctx context.Context, payload *Order) error {
				fmt.Printf("[save]")
				return nil
			},
		},
		StateCancelled: {
			Save: func(ctx context.Context, payload *Order) error {
				fmt.Printf("[save]")
				return nil
			},
		},
	},
}

// --- конвертер: восстанавливает текущее состояние из payload ---

type conv struct{}

func (c *conv) CurrentState(_ context.Context, o *Order) (State, error) {
	return StatePending, nil
}

// --- колбэки ---

type logger struct{}

func (l *logger) Before(_ context.Context, state State) error {
	fmt.Printf("→ входим в состояние %q\n", state)
	return nil
}

func (l *logger) After(_ context.Context, event Event, next State) error {
	fmt.Printf("← событие %q → переход в %q\n\n", event, next)
	return nil
}

// --- main ---

func main() {
	order := &Order{
		ID:        42,
		UserEmail: "user@example.com",
		Amount:    1500.00,
	}

	fmt.Println("=== запуск машины состояний ===")
	m, err := statety.NewMachine[State, Event, *Order](config, &logger{}, &conv{})
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	result, err := m.Work(context.Background(), order)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}

	switch result {
	case statety.Final:
		fmt.Printf("=== заказ #%d доставлен (трек: %s) ===\n", order.ID, order.TrackingNum)
	case statety.Stop:
		fmt.Printf("=== заказ #%d отменён ===\n", order.ID)
	}

	fmt.Println(statety.DOT(config))
}
