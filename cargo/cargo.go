package cargo

import (
	"errors"
	"strings"

	"github.com/Qalifah/shipping/location"
	"github.com/pborman/uuid"
)

// TrackingID uniquely identifies a cargo
type TrackingID string

// Cargo contains info about a cargo
type Cargo struct {
	TrackingID TrackingID
	Origin		location.UNLcode
	RouteSpecification	RouteSpecification
	Itinerary 		Itinerary
	Delivery		Delivery
}

// SpecifyNewRoute specifies a new route for this cargo
func(c *Cargo) SpecifyNewRoute(rs RouteSpecification) {
	c.RouteSpecification = rs
	c.Delivery = c.Delivery.UpdateOnRouting(c.RouteSpecification, c.Itinerary)
}

// AssignToRoute attachs a new itinerary to the cargo
func(c *Cargo) AssignToRoute(itinerary Itinerary) {
	c.Itinerary = itinerary
	c.Delivery = c.Delivery.UpdateOnRouting(c.RouteSpecification, c.Itinerary)
}

// DeriveDeliveryProgress updates all aspects of the cargo aggregate status
// based on the current route specification, itinerary and handling of the cargo
func(c *Cargo) DeriveDeliveryProgress(history HandlingHistory) {
	c.Delivery = DeriveDeliveryFrom(c.RouteSpecification, c.Itinerary, history)
}

// New creates a new, unrouted cargo
func New(id TrackingID, rs RouteSpecification) *Cargo {
	itinerary := Itinerary{}
	history := HandlingHistory{make([]HandlingEvent, 0)}

	return &Cargo{
		TrackingID:         id,
		Origin:             rs.Origin,
		RouteSpecification: rs,
		Delivery:           DeriveDeliveryFrom(rs, itinerary, history),
	}
}

// Repository provides access to cargo store
type Repository interface {
	Store(cargo *Cargo) error
	Find(id TrackingID) (*Cargo, error)
	FindAll() []*Cargo
}

// ErrUnknown is used when a cargo can't be found
var ErrUnknown = errors.New("unknown cargo")

// NextTrackingID generates a new tracking ID.
func NextTrackingID() TrackingID {
	return TrackingID(strings.Split(strings.ToUpper(uuid.New()), "-")[0])
}