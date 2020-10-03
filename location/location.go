package location

import "errors"

// UNLcode uniquely identifies a location
type UNLcode string

// Location represents a location of a cargo
type Location struct {
	UNLcode UNLcode
	Name	string
}

// ErrUnknown is used when a location can't be found
var ErrUnknown = errors.New("unknown location")

// Repository represents a location store
type Repository interface {
	Find(UNLcode) (*Location, error)
	FindAll() []*Location
}