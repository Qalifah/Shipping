package booking

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"

	"github.com/Qalifah/shipping/cargo"
	"github.com/Qalifah/shipping/location"
	pb "github.com/Qalifah/shipping/pb/bookingpb"
	"github.com/Qalifah/shipping/voyage"

	"github.com/golang/protobuf/ptypes"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/go-kit/kit/tracing/zipkin"
	"github.com/go-kit/kit/transport"
	grpctransport "github.com/go-kit/kit/transport/grpc"
)

type grpcServer struct {
	bookCargo         grpctransport.Handler
	loadCargo         grpctransport.Handler
	requestRoutes     grpctransport.Handler
	assignRoute       grpctransport.Handler
	changeDestination grpctransport.Handler
	listCargos        grpctransport.Handler
	listLocations     grpctransport.Handler
}

// NewGRPCServer makes a set of endpoints available on a grpc server
func NewGRPCServer(endpoints Set, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) pb.BookingServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
	}

	if zipkinTracer != nil {
		options = append(options, zipkin.GRPCServerTrace(zipkinTracer))
	}

	return &grpcServer{
		bookCargo: grpctransport.NewServer(
			endpoints.BookCargoEndpoint,
			decodeGRPCBookCargoRequest,
			encodeGRPCBookCargoResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "bookCargo", logger)))...,
		),

		loadCargo: grpctransport.NewServer(
			endpoints.LoadCargoEndpoint,
			decodeGRPCLoadCargoRequest,
			encodeGRPCLoadCargoResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "loadCargo", logger)))...,
		),

		requestRoutes: grpctransport.NewServer(
			endpoints.RequestRoutesEndpoint,
			decodeGRPCRoutesForCargoRequest,
			encodeGRPCBookCargoResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "requestRoutes", logger)))...,
		),

		assignRoute: grpctransport.NewServer(
			endpoints.AssignRouteEndpoint,
			decodeGRPCBookCargoRequest,
			encodeGRPCBookCargoResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "assignRoute", logger)))...,
		),

		changeDestination: grpctransport.NewServer(
			endpoints.ChangeDestinationEndpoint,
			decodeGRPCChangeDestinationRequest,
			encodeGRPCChangeDestinationResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "changeDestination", logger)))...,
		),

		listCargos: grpctransport.NewServer(
			endpoints.ListCargosEndpoint,
			decodeGRPCCargosRequest,
			encodeGRPCCargosResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "listCargos", logger)))...,
		),

		listLocations: grpctransport.NewServer(
			endpoints.ListLocationsEndpoint,
			decodeGRPCLocationsRequest,
			encodeGRPCLocationsResponse,
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(otTracer, "listLocations", logger)))...,
		),
	}
}

func (s *grpcServer) BookNewCargo(ctx context.Context, req *pb.NewCargoRequest) (*pb.NewCargoReply, error) {
	_, rep, err := s.bookCargo.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	return rep.(*pb.NewCargoReply), nil
}

func (s *grpcServer) LoadCargo(ctx context.Context, req *pb.LoadCargoRequest) (*pb.LoadCargoReply, error) {
	_, rep, err := s.loadCargo.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	return rep.(*pb.LoadCargoReply), nil
}

func (s *grpcServer) RequestPossibleRoutesForCargo(ctx context.Context, req *pb.RoutesForCargoRequest) (*pb.RoutesForCargoReply, error) {
	_, rep, err := s.requestRoutes.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	return rep.(*pb.RoutesForCargoReply), nil
}

func (s *grpcServer) AssignCargoToRoute(ctx context.Context, req *pb.CargoToRouteRequest) (*pb.CargoToRouteReply, error) {
	_, rep, err := s.assignRoute.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	return rep.(*pb.CargoToRouteReply), nil
}

func (s *grpcServer) ChangeDestination(ctx context.Context, req *pb.ChangeDestinationRequest) (*pb.ChangeDestinationReply, error) {
	_, rep, err := s.changeDestination.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	return rep.(*pb.ChangeDestinationReply), nil
}

func (s *grpcServer) Cargos(ctx context.Context, req *pb.CargosRequest) (*pb.CargosReply, error) {
	_, rep, err := s.listCargos.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	return rep.(*pb.CargosReply), nil
}

func (s *grpcServer) Locations(ctx context.Context, req *pb.LocationsRequest) (*pb.LocationsReply, error) {
	_, rep, err := s.listLocations.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	return rep.(*pb.LocationsReply), nil
}

// NewGRPCClient returns a booking service backed by a grpc server at the other end of the conn
func NewGRPCClient(conn *grpc.ClientConn, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) Service {
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))
	var options []grpctransport.ClientOption
	if zipkinTracer != nil {
		options = append(options, zipkin.GRPCClientTrace(zipkinTracer))
	}
	var bookCargoEndpoint endpoint.Endpoint
	{
		bookCargoEndpoint = grpctransport.NewClient(
			conn,
			"pb.Booking",
			"BookCargo",
			encodeGRPCBookCargoRequest,
			decodeGRPCBookCargoResponse,
			pb.NewCargoReply{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		bookCargoEndpoint = opentracing.TraceClient(otTracer, "Book Cargo")(bookCargoEndpoint)
		bookCargoEndpoint = limiter(bookCargoEndpoint)
		bookCargoEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Book Cargo",
			Timeout: 30 * time.Second,
		}))(bookCargoEndpoint)
	}

	var loadCargoEndpoint endpoint.Endpoint
	{
		loadCargoEndpoint = grpctransport.NewClient(
			conn,
			"pb.Booking",
			"LoadCargo",
			encodeGRPCLoadCargoRequest,
			decodeGRPCLoadCargoResponse,
			pb.LoadCargoReply{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		loadCargoEndpoint = opentracing.TraceClient(otTracer, "Load Cargo")(loadCargoEndpoint)
		loadCargoEndpoint = limiter(loadCargoEndpoint)
		loadCargoEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Load Cargo",
			Timeout: 30 * time.Second,
		}))(loadCargoEndpoint)
	}

	var requestRoutesEndpoint endpoint.Endpoint
	{
		requestRoutesEndpoint = grpctransport.NewClient(
			conn,
			"pb.Booking",
			"RequestRoutes",
			encodeGRPCRoutesForCargoRequest,
			decodeGRPCRoutesForCargoResponse,
			pb.RoutesForCargoReply{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		requestRoutesEndpoint = opentracing.TraceClient(otTracer, "Request Possible Cargo Routes")(requestRoutesEndpoint)
		requestRoutesEndpoint = limiter(requestRoutesEndpoint)
		requestRoutesEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Request Possible Cargo Routes",
			Timeout: 30 * time.Second,
		}))(requestRoutesEndpoint)
	}

	var assignRouteEndpoint endpoint.Endpoint
	{
		assignRouteEndpoint = grpctransport.NewClient(
			conn,
			"pb.Booking",
			"AssignRoute",
			encodeGRPCCargoToRouteRequest,
			decodeGRPCCargoToRouteResponse,
			pb.CargoToRouteReply{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		assignRouteEndpoint = opentracing.TraceClient(otTracer, "Assign Route to Cargo")(assignRouteEndpoint)
		assignRouteEndpoint = limiter(assignRouteEndpoint)
		assignRouteEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Assign Route to Cargo",
			Timeout: 30 * time.Second,
		}))(assignRouteEndpoint)
	}

	var changeDestinationEndpoint endpoint.Endpoint
	{
		changeDestinationEndpoint = grpctransport.NewClient(
			conn,
			"pb.Booking",
			"ChangeDestination",
			encodeGRPCChangeDestinationRequest,
			decodeGRPCChangeDestinationResponse,
			pb.ChangeDestinationReply{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		changeDestinationEndpoint = opentracing.TraceClient(otTracer, "Change Destination")(changeDestinationEndpoint)
		changeDestinationEndpoint = limiter(changeDestinationEndpoint)
		changeDestinationEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Change Destination",
			Timeout: 30 * time.Second,
		}))(changeDestinationEndpoint)
	}

	var listCargosEndpoint endpoint.Endpoint
	{
		listCargosEndpoint = grpctransport.NewClient(
			conn,
			"pb.Booking",
			"Cargos",
			encodeGRPCCargosRequest,
			decodeGRPCCargosResponse,
			pb.CargosReply{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		listCargosEndpoint = opentracing.TraceClient(otTracer, "Cargos")(listCargosEndpoint)
		listCargosEndpoint = limiter(listCargosEndpoint)
		listCargosEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Cargos",
			Timeout: 30 * time.Second,
		}))(listCargosEndpoint)
	}

	var listLocationsEndpoint endpoint.Endpoint
	{
		listLocationsEndpoint = grpctransport.NewClient(
			conn,
			"pb.Booking",
			"Locations",
			encodeGRPCLocationsRequest,
			decodeGRPCLocationsResponse,
			pb.LocationsReply{},
			append(options, grpctransport.ClientBefore(opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		listLocationsEndpoint = opentracing.TraceClient(otTracer, "Locations")(listLocationsEndpoint)
		listLocationsEndpoint = limiter(listLocationsEndpoint)
		listCargosEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Locations",
			Timeout: 30 * time.Second,
		}))(listLocationsEndpoint)
	}

	return Set{
		BookCargoEndpoint:         bookCargoEndpoint,
		LoadCargoEndpoint:         loadCargoEndpoint,
		RequestRoutesEndpoint:     requestRoutesEndpoint,
		AssignRouteEndpoint:       assignRouteEndpoint,
		ChangeDestinationEndpoint: changeDestinationEndpoint,
		ListCargosEndpoint:        listCargosEndpoint,
		ListLocationsEndpoint:     listLocationsEndpoint,
	}
}

func decodeGRPCBookCargoRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.NewCargoRequest)
	deadline, _ := ptypes.Timestamp(req.Deadline)
	return bookCargoRequest{
		Origin:          location.UNLcode(req.Origin),
		Destination:     location.UNLcode(req.Destination),
		ArrivalDeadline: deadline,
	}, nil
}

func decodeGRPCLoadCargoRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.LoadCargoRequest)
	return loadCargoRequest{
		ID: cargo.TrackingID(req.TrackingId),
	}, nil
}

func decodeGRPCRoutesForCargoRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.RoutesForCargoRequest)
	return requestRoutesRequest{
		ID: cargo.TrackingID(req.TrackingId),
	}, nil
}

func decodeGRPCCargoToRouteRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.CargoToRouteRequest)
	itinerary := cargo.Itinerary{
		Legs: decodeLegs(req.Itinerary.Legs),
	}
	return assignRouteRequest{
		ID:        cargo.TrackingID(req.TrackingId),
		Itinerary: itinerary,
	}, nil
}

func decodeGRPCChangeDestinationRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.ChangeDestinationRequest)
	return changeDestinationRequest{
		ID:          cargo.TrackingID(req.TrackingId),
		Destination: location.UNLcode(req.Destination),
	}, nil
}

func decodeGRPCCargosRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	_ = grpcReq.(*pb.CargosRequest)
	return listCargosRequest{}, nil
}

func decodeGRPCLocationsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	_ = grpcReq.(*pb.LocationsRequest)
	return listLocationsRequest{}, nil
}

func encodeGRPCBookCargoResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(bookCargoResponse)
	return &pb.NewCargoReply{
		TrackingId: string(resp.ID),
		Err:        err2str(resp.Err),
	}, nil
}

func encodeGRPCLoadCargoResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(loadCargoResponse)
	return &pb.LoadCargoReply{
		Cargo: encodeCargo(*resp.Cargo),
		Err:   err2str(resp.Err),
	}, nil
}

func encodeGRPCRoutesForCargoResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(requestRoutesResponse)
	var itineraries []*pb.Itinerary
	for _, route := range resp.Routes {
		itinerary := &pb.Itinerary{
			Legs: encodeLegs(route.Legs),
		}
		itineraries = append(itineraries, itinerary)
	}
	return &pb.RoutesForCargoReply{
		Itineraries: itineraries,
	}, nil
}

func encodeGRPCCargoToRouteResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(assignRouteResponse)
	return &pb.CargoToRouteReply{Err: err2str(resp.Err)}, nil
}

func encodeGRPCChangeDestinationResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(changeDestinationResponse)
	return &pb.ChangeDestinationReply{Err: err2str(resp.Err)}, nil
}

func encodeGRPCCargosResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(listCargosResponse)
	var cargos []*pb.Cargo
	for _, cargo := range resp.Cargos {
		temp := encodeCargo(cargo)
		cargos = append(cargos, temp)
	}
	return &pb.CargosReply{Cargos: cargos}, nil
}

func encodeGRPCLocationsResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(listLocationsResponse)
	var locations []*pb.Location
	for _, location := range resp.Locations {
		locations = append(locations, encodeLocation(location))
	}
	return &pb.LocationsReply{Locations: locations}, nil

}

func encodeGRPCBookCargoRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(bookCargoRequest)
	arrivalDeadline, _ := ptypes.TimestampProto(req.ArrivalDeadline)
	return &pb.NewCargoRequest{
		Origin:      string(req.Origin),
		Destination: string(req.Destination),
		Deadline:    arrivalDeadline,
	}, nil
}

func encodeGRPCLoadCargoRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(loadCargoRequest)
	return &pb.LoadCargoRequest{TrackingId: string(req.ID)}, nil
}

func encodeGRPCRoutesForCargoRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(requestRoutesRequest)
	return &pb.RoutesForCargoRequest{TrackingId: string(req.ID)}, nil
}

func encodeGRPCCargoToRouteRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(assignRouteRequest)
	return &pb.CargoToRouteRequest{
		TrackingId: string(req.ID),
		Itinerary:  &pb.Itinerary{Legs: encodeLegs(req.Itinerary.Legs)},
	}, nil
}

func encodeGRPCChangeDestinationRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(changeDestinationRequest)
	return &pb.ChangeDestinationRequest{TrackingId: string(req.ID), Destination: string(req.Destination)}, nil
}

func encodeGRPCCargosRequest(_ context.Context, request interface{}) (interface{}, error) {
	_ = request.(listCargosRequest)
	return &pb.CargosRequest{}, nil
}

func encodeGRPCLocationsRequest(_ context.Context, request interface{}) (interface{}, error) {
	_ = request.(listLocationsRequest)
	return &pb.LocationsRequest{}, nil
}

func decodeGRPCBookCargoResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.NewCargoReply)
	return bookCargoResponse{ID: cargo.TrackingID(reply.TrackingId), Err: str2err(reply.Err)}, nil
}

func decodeGRPCLoadCargoResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.LoadCargoReply)
	return loadCargoResponse{Cargo: decodeCargo(reply.Cargo), Err: str2err(reply.Err)}, nil
}

func decodeGRPCRoutesForCargoResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.RoutesForCargoReply)
	var itineraries []cargo.Itinerary
	for _, itinerary := range reply.Itineraries {
		temp := cargo.Itinerary{
			Legs: decodeLegs(itinerary.Legs),
		}
		itineraries = append(itineraries, temp)
	}
	return requestRoutesResponse{
		Routes: itineraries,
		Err:    nil,
	}, nil
}

func decodeGRPCCargoToRouteResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.CargoToRouteReply)
	return assignRouteResponse{Err: str2err(reply.Err)}, nil
}

func decodeGRPCChangeDestinationResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.ChangeDestinationReply)
	return changeDestinationResponse{Err: str2err(reply.Err)}, nil
}

func decodeGRPCCargosResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.CargosReply)
	var cargos []Cargo
	for _, cargo := range reply.Cargos {
		// decoding the cargo returns a pointer of the booking's cargo, but the cargo is what's needed not the pointer
		// so to reference the cargo itself i added '*' in front.
		cargos = append(cargos, *decodeCargo(cargo))
	}
	return listCargosResponse{Cargos: cargos, Err: nil}, nil
}

func decodeGRPCLocationsResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.LocationsReply)
	var locations []Location
	for _, location := range reply.Locations {
		locations = append(locations, decodeLocation(location))
	}
	return listLocationsResponse{Locations: locations, Err: nil}, nil
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func encodeLegs(decodedLegs []cargo.Leg) []*pb.Leg {
	var encodedLegs []*pb.Leg
	for _, leg := range decodedLegs {
		loadTime, _ := ptypes.TimestampProto(leg.LoadTime)
		unloadTime, _ := ptypes.TimestampProto(leg.UnLoadTime)
		leg := &pb.Leg{
			VoyageNumber:   string(leg.VoyageNumber),
			LoadLocation:   string(leg.LoadLocation),
			UnloadLocation: string(leg.UnLoadLocation),
			LoadTime:       loadTime,
			UnloadTime:     unloadTime,
		}
		encodedLegs = append(encodedLegs, leg)
	}
	return encodedLegs
}

func decodeLegs(encodedLegs []*pb.Leg) []cargo.Leg {
	var decodedLegs []cargo.Leg
	for _, i := range encodedLegs {
		loadTime, _ := ptypes.Timestamp(i.LoadTime)
		unloadTime, _ := ptypes.Timestamp(i.UnloadTime)
		temp := cargo.Leg{
			VoyageNumber:   voyage.Number(i.VoyageNumber),
			LoadLocation:   location.UNLcode(i.LoadLocation),
			UnLoadLocation: location.UNLcode(i.UnloadLocation),
			LoadTime:       loadTime,
			UnLoadTime:     unloadTime,
		}
		decodedLegs = append(decodedLegs, temp)
	}
	return decodedLegs
}

func encodeCargo(decodedCargo Cargo) *pb.Cargo {
	arrivalDeadline, _ := ptypes.TimestampProto(decodedCargo.ArrivalDeadline)
	encodedCargo := &pb.Cargo{
		ArrivalDeadline: arrivalDeadline,
		Destination:     decodedCargo.Destination,
		Legs:            encodeLegs(decodedCargo.Legs),
		Misrouted:       decodedCargo.Misrouted,
		Origin:          decodedCargo.Origin,
		Routed:          decodedCargo.Routed,
		TrackingId:      decodedCargo.TrackingID,
	}
	return encodedCargo
}

func decodeCargo(encodedCargo *pb.Cargo) *Cargo {
	arrivalDeadline, _ := ptypes.Timestamp(encodedCargo.ArrivalDeadline)
	decodedCargo := &Cargo{
		ArrivalDeadline: arrivalDeadline,
		Destination:     encodedCargo.Destination,
		Legs:            decodeLegs(encodedCargo.Legs),
		Misrouted:       encodedCargo.Misrouted,
		Origin:          encodedCargo.Origin,
		Routed:          encodedCargo.Routed,
		TrackingID:      encodedCargo.TrackingId,
	}
	return decodedCargo
}

func encodeLocation(decodedLocation Location) *pb.Location {
	encodedLocation := &pb.Location{
		Unlcode: decodedLocation.UNLcode,
		Name:    decodedLocation.Name,
	}
	return encodedLocation
}

func decodeLocation(encodedLocation *pb.Location) Location {
	decodedLocation := Location{
		UNLcode: encodedLocation.Unlcode,
		Name:    encodedLocation.Name,
	}
	return decodedLocation
}
