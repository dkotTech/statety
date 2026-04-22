package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/dkotTech/statety"
)

type (
	State string
	Event string

	Order struct {
		sync.Mutex
	}
)

const (
	// states
	StateCreated           State = "created"
	StateFraudCheck        State = "fraud_check"
	StateFraudBlocked      State = "fraud_blocked"
	StatePaymentPending    State = "payment_pending"
	StatePaymentProcessing State = "payment_processing"
	StatePaymentFailed     State = "payment_failed"
	StateConfirmed         State = "confirmed"
	StateWarehousePicking  State = "warehouse_picking"
	StateWarehousePacked   State = "warehouse_packed"
	StateAwaitingStock     State = "awaiting_stock"
	StateReadyToShip       State = "ready_to_ship"
	StateInTransit         State = "in_transit"
	StateCustomsHold       State = "customs_hold"
	StateOutForDelivery    State = "out_for_delivery"
	StateDeliveryFailed    State = "delivery_failed"
	StateDelivered         State = "delivered"
	StateReturnRequested   State = "return_requested"
	StateReturnInTransit   State = "return_in_transit"
	StateReturnReceived    State = "return_received"
	StateRefundProcessing  State = "refund_processing"
	StateRefunded          State = "refunded"
	StateDispute           State = "dispute"
	StateDisputeResolved   State = "dispute_resolved"
	StateCancelled         State = "cancelled"

	// events
	EventFraudPassed      Event = "fraud_passed"
	EventFraudFailed      Event = "fraud_failed"
	EventPaymentInitiated Event = "payment_initiated"
	EventPaymentSuccess   Event = "payment_success"
	EventPaymentDeclined  Event = "payment_declined"
	EventPaymentRetry     Event = "payment_retry"
	EventPaymentTimeout   Event = "payment_timeout"
	EventStockAvailable   Event = "stock_available"
	EventStockUnavailable Event = "stock_unavailable"
	EventPickingDone      Event = "picking_done"
	EventPacked           Event = "packed"
	EventShipped          Event = "shipped"
	EventCustomsCleared   Event = "customs_cleared"
	EventCustomsRejected  Event = "customs_rejected"
	EventOutForDelivery   Event = "out_for_delivery"
	EventDelivered        Event = "delivered"
	EventDeliveryFailed   Event = "delivery_failed"
	EventDeliveryRetry    Event = "delivery_retry"
	EventReturnRequested  Event = "return_requested"
	EventReturnShipped    Event = "return_shipped"
	EventReturnArrived    Event = "return_arrived"
	EventRefundApproved   Event = "refund_approved"
	EventRefundCompleted  Event = "refund_completed"
	EventDisputeOpened    Event = "dispute_opened"
	EventDisputeResolved  Event = "dispute_resolved"
	EventCancelled        Event = "cancelled"
)

var noop = func(_ context.Context, _ *Order) (Event, error) { return "", nil }
var save = func(_ context.Context, _ *Order) error { return nil }

var config = statety.Setup[State, Event, *Order]{
	StartState: StateCreated,

	FinalStates: []State{
		StateDelivered,
		StateRefunded,
		StateDisputeResolved,
		StateCancelled,
		StateFraudBlocked,
	},

	Config: map[State]statety.Steps[State, Event, *Order]{
		StateCreated: {
			Do:   noop,
			Next: map[Event]State{EventFraudPassed: StateFraudCheck, EventCancelled: StateCancelled},
		},
		StateFraudCheck: {
			Do: noop,
			Next: map[Event]State{
				EventFraudPassed: StatePaymentPending,
				EventFraudFailed: StateFraudBlocked,
			},
		},
		StateFraudBlocked: {
			Do:   noop,
			Next: map[Event]State{},
		},
		StatePaymentPending: {
			Do: noop,
			Next: map[Event]State{
				EventPaymentInitiated: StatePaymentProcessing,
				EventCancelled:        StateCancelled,
			},
		},
		StatePaymentProcessing: {
			Do: noop,
			Next: map[Event]State{
				EventPaymentSuccess:  StateConfirmed,
				EventPaymentDeclined: StatePaymentFailed,
				EventPaymentTimeout:  StatePaymentPending,
			},
		},
		StatePaymentFailed: {
			Do: noop,
			Next: map[Event]State{
				EventPaymentRetry: StatePaymentProcessing,
				EventCancelled:    StateCancelled,
			},
		},
		StateConfirmed: {
			Do:   noop,
			Save: save,
			Next: map[Event]State{
				EventStockAvailable:   StateWarehousePicking,
				EventStockUnavailable: StateAwaitingStock,
				EventCancelled:        StateCancelled,
			},
		},
		StateAwaitingStock: {
			Do: noop,
			Next: map[Event]State{
				EventStockAvailable: StateWarehousePicking,
				EventCancelled:      StateCancelled,
			},
		},
		StateWarehousePicking: {
			Do: noop,
			Next: map[Event]State{
				EventPickingDone: StateWarehousePacked,
			},
		},
		StateWarehousePacked: {
			Do:   noop,
			Save: save,
			Next: map[Event]State{
				EventShipped: StateReadyToShip,
			},
		},
		StateReadyToShip: {
			Do:   noop,
			Save: save,
			Next: map[Event]State{
				EventShipped: StateInTransit,
			},
		},
		StateInTransit: {
			Do: noop,
			Next: map[Event]State{
				EventCustomsCleared:  StateOutForDelivery,
				EventCustomsRejected: StateCustomsHold,
				EventOutForDelivery:  StateOutForDelivery,
			},
		},
		StateCustomsHold: {
			Do: noop,
			Next: map[Event]State{
				EventCustomsCleared:  StateInTransit,
				EventCustomsRejected: StateCancelled,
			},
		},
		StateOutForDelivery: {
			Do:   noop,
			Save: save,
			Next: map[Event]State{
				EventDelivered:      StateDelivered,
				EventDeliveryFailed: StateDeliveryFailed,
			},
		},
		StateDeliveryFailed: {
			Do: noop,
			Next: map[Event]State{
				EventDeliveryRetry: StateOutForDelivery,
			},
		},
		StateDelivered: {
			Do:   noop,
			Save: save,
			Next: map[Event]State{
				EventReturnRequested: StateReturnRequested,
				EventDisputeOpened:   StateDispute,
			},
		},
		StateReturnRequested: {
			Do: noop,
			Next: map[Event]State{
				EventReturnShipped: StateReturnInTransit,
				EventCancelled:     StateDelivered,
			},
		},
		StateReturnInTransit: {
			Do: noop,
			Next: map[Event]State{
				EventReturnArrived: StateReturnReceived,
			},
		},
		StateReturnReceived: {
			Do:   noop,
			Save: save,
			Next: map[Event]State{
				EventRefundApproved: StateRefundProcessing,
			},
		},
		StateRefundProcessing: {
			Do: noop,
			Next: map[Event]State{
				EventRefundCompleted: StateRefunded,
			},
		},
		StateRefunded: {
			Do:   noop,
			Save: save,
			Next: map[Event]State{},
		},
		StateDispute: {
			Do: noop,
			Next: map[Event]State{
				EventDisputeResolved: StateDisputeResolved,
				EventRefundApproved:  StateRefundProcessing,
			},
		},
		StateDisputeResolved: {
			Do:   noop,
			Next: map[Event]State{},
		},
		StateCancelled: {
			Do:   noop,
			Save: save,
			Next: map[Event]State{},
		},
	},
}

func main() {
	fmt.Println(statety.DOT(config))
}
