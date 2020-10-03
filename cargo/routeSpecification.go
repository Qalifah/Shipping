package cargo

import (
	"time"

	"github.com/Qalifah/shipping/location"
)

// RouteSpecification gives details about the movement of a cargo
type RouteSpecification struct {
	Origin location.UNLcode
	Destination	location.UNLcode
	Deadline	time.Time
}

// IsSatisfiedBy checks whether itinerary satisfies this specification
func (s RouteSpecification) IsSatisfiedBy(itinerary Itinerary) bool {
	return itinerary.Legs != nil && s.Origin == itinerary.InitialDepartureLocation() && s.Destination == itinerary.FinalArrivalLocation()
}

// RoutingStatus describes the status of a cargo routing
type RoutingStatus int

// valid routing statuses
const (
	NotRouted RoutingStatus = iota
	MisRouted
	Routed
)

func(s RoutingStatus) String() string {
	switch s {
	case NotRouted:
		return "Not routed"
	case MisRouted:
		return	"Misrouted"
	case Routed:
		return "Routed"
	}
	return ""
}

// TransportStatus describes the status of a cargo transportation
type TransportStatus int

// Valid transport statuses
const (
	NotReceived	TransportStatus = iota
	InPort
	OnboardCarrier
	Claimed
	Unknown
)

func(s TransportStatus) String() string {
	switch s {
	case NotReceived:
		return "Not Received"
	case InPort:
		return	"In Port"
	case OnboardCarrier:
		return "Onboard Carrier"
	case Claimed:
		return "Claimed"
	case Unknown:
		return "Unknown"
	}
	return ""
}