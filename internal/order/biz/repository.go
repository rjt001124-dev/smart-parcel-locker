package biz

import "context"

// Repository is the persistence contract for orders. Implementations must keep
// order mutation, locker-cell mutation, and status-log append atomic (one
// transaction per transition).
type Repository interface {
	// CreateOrder persists a new order and locks an idle cell of the requested
	// size at the site. The returned order carries the generated id and the
	// locked cell identifiers. Returns ErrNoAvailableCell when no cell can be
	// reserved, or ErrSiteNotFound when the site does not exist.
	CreateOrder(context.Context, *Order) (*Order, error)
	// GetOrder loads an order with site/device/cell names joined in.
	GetOrder(context.Context, uint64) (*Order, error)
	// ListOrders returns every order for the user, newest first.
	ListOrders(context.Context, string) ([]*Order, error)
	// MarkOverdue transitions an IN_STORAGE order to OVERDUE with the given
	// overdue fee. It is idempotent when the order has already left IN_STORAGE.
	MarkOverdue(context.Context, uint64, int64) error
	// CancelOrder moves PENDING_PAYMENT -> CANCELLED and releases the cell.
	CancelOrder(context.Context, uint64) (*Order, error)
	// OpenDepositDoor moves AWAITING_DEPOSIT -> IN_STORAGE and occupies the cell.
	OpenDepositDoor(context.Context, uint64) (*Order, error)
	// OpenPickupDoor moves IN_STORAGE (not overdue) or AWAITING_PICKUP ->
	// COMPLETED and releases the cell.
	OpenPickupDoor(context.Context, uint64) (*Order, error)
	// ConfirmDevelopmentPayment moves PENDING_PAYMENT -> PAID -> AWAITING_DEPOSIT.
	ConfirmDevelopmentPayment(context.Context, uint64) (*Order, error)
	// SettleOverdue moves OVERDUE -> AWAITING_PICKUP and clears the overdue fee.
	SettleOverdue(context.Context, uint64) (*Order, error)
}

// DoorGateway abstracts the device gateway that physically opens a locker door.
// The milestone build provides a simulator implementation; a production build
// would route this through the device command domain.
type DoorGateway interface {
	OpenDepositDoor(context.Context, string) error
	OpenPickupDoor(context.Context, string) error
}
