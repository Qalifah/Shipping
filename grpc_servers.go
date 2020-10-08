// still under development, yet to be completed
package main

import (
	"github.com/Qalifah/shipping/pb/bookingpb"
	"github.com/Qalifah/shipping/pb/handlingpb"
	"github.com/Qalifah/shipping/pb/trackingpb"
	"github.com/Qalifah/shipping/booking"
	"github.com/Qalifah/shipping/handling"
	"github.com/Qalifah/shipping/tracking"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"

	"github.com/go-kit/kit/log"
)

// GRPCServers provides access to the grpcservers in our application
type gRPCServers struct {
	bookingpb.BookingServer
	handlingpb.HandlingServer
	trackingpb.TrackingServer
}

// NewgRPCServers creates a new instance of GRPCServers
func NewgRPCServers(bookingSet booking.Set, handlingSet handling.Set, trackingSet tracking.Set, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) gRPCServers {
	return gRPCServers{
		booking.NewGRPCServer(bookingSet, otTracer, zipkinTracer, logger),
		handling.NewGRPCServer(handlingSet, otTracer, zipkinTracer, logger),
		tracking.NewGRPCServer(trackingSet, otTracer, zipkinTracer, logger),
	}
}