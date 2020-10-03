package voyage

import (
	"errors"
	"time"

	"github.com/Qalifah/shipping/location"
)

// Number uniquely identifies a voyage
type Number string

// Voyage is a uniquely identifiable series of carrier movements
type Voyage struct {
	Number Number
	Schedule Schedule
}

// New creates a new instance of a voyage
func New(n Number, s Schedule) *Voyage {
	return &Voyage{Number: n, Schedule: s}
}

// Schedule describes a voyage schedule
type Schedule struct {
	CarrierMovements	[]CarrierMovement
}

// CarrierMovement is a vessel voyage from one location to another
type CarrierMovement struct {
	DepartureLocation	location.UNLcode
	ArrivalLocation		location.UNLcode
	DepartureTime		time.Time
	ArrivalTime			time.Time
}

// ErrUnknown is used when a voyage can't be found
var ErrUnknown = errors.New("unknown voyage")

// Repository provides access to a voyage store
type Repository interface {
	Find(Number) (*Voyage, error)
}
