package routing

import (
	"github.com/Qalifah/shipping/cargo"
)

// Service provides access to an external routing service.
type Service interface {
	// FetchRoutesForSpecification finds all possible routes that satisfy a
	// given specification.
	FetchRoutesForSpecification(rs cargo.RouteSpecification) []cargo.Itinerary
}