package flight

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/flight"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type FlightServer struct {
	flightServer flight.Server
	mem         memory.Allocator
	port        int
	listener    net.Listener
	mu          sync.Mutex
}

func NewFlightServer(port int) *FlightServer {
	return &FlightServer{
		mem:  memory.NewGoAllocator(),
		port: port,
	}
}

func (s *FlightServer) Start(ctx context.Context) error {
	s.mu.Lock()
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to bind Arrow Flight listener on %s: %w", addr, err)
	}
	s.listener = listener
	s.mu.Unlock()

	s.flightServer = flight.NewServerWithMiddleware(nil)
	flightService := &flightServiceServer{mem: s.mem}
	s.flightServer.RegisterFlightService(flightService)

	go func() {
		<-ctx.Done()
		if s.flightServer != nil {
			s.flightServer.Shutdown()
		}
	}()

	log.Printf("[FlightSQL] Arrow Flight gRPC server starting on port %d...", s.port)
	if err := s.flightServer.Init(addr); err != nil {
		return fmt.Errorf("failed to init flight server: %w", err)
	}
	go s.flightServer.Serve()
	return nil
}

func (s *FlightServer) Stop() {
	if s.flightServer != nil {
		s.flightServer.Shutdown()
	}
}

type flightServiceServer struct {
	flight.BaseFlightServer
	mem memory.Allocator
}

func (s *flightServiceServer) DoGet(ticket *flight.Ticket, stream flight.FlightService_DoGetServer) error {
	ctx := stream.Context()

	tenantID, err := extractTenantIDFromGRPC(ctx)
	if err != nil {
		return fmt.Errorf("unauthenticated: %w", err)
	}

	schema := BuildPortfolioSchema()
	b := array.NewRecordBuilder(s.mem, schema)

	b.Field(0).(*array.StringBuilder).Append(tenantID.String())
	b.Field(1).(*array.StringBuilder).Append("PT-HOLDING-001")
	b.Field(2).(*array.StringBuilder).Append("AAPL")
	b.Field(3).(*array.Float64Builder).Append(2100000.0)
	b.Field(4).(*array.Float64Builder).Append(0.210)

	rec := b.NewRecord()
	b.Release()
	defer rec.Release()

	writer := flight.NewRecordWriter(stream, ipc.WithSchema(schema))
	defer writer.Close()

	return writer.Write(rec)
}

func (s *flightServiceServer) DoPut(stream flight.FlightService_DoPutServer) error {
	return nil
}

func (s *flightServiceServer) ListFlights(criteria *flight.Criteria, stream flight.FlightService_ListFlightsServer) error {
	return nil
}

func extractTenantIDFromGRPC(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, fmt.Errorf("missing gRPC metadata")
	}

	tenantHeaders := md.Get("x-tenant-id")
	if len(tenantHeaders) == 0 || tenantHeaders[0] == "" {
		return uuid.Nil, fmt.Errorf("missing x-tenant-id gRPC header")
	}

	return uuid.Parse(tenantHeaders[0])
}

func BuildPortfolioSchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{
		{Name: "tenant_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "portfolio_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "security_isin", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "market_value", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
		{Name: "effective_exposure_pct", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	}, nil)
}

func BuildSamplePortfolioRecord(mem memory.Allocator, tenantID, portfolioID string) arrow.Record {
	schema := BuildPortfolioSchema()
	b := array.NewRecordBuilder(mem, schema)
	defer b.Release()

	b.Field(0).(*array.StringBuilder).Append(tenantID)
	b.Field(1).(*array.StringBuilder).Append(portfolioID)
	b.Field(2).(*array.StringBuilder).Append("AAPL")
	b.Field(3).(*array.Float64Builder).Append(2100000.0)
	b.Field(4).(*array.Float64Builder).Append(0.210)

	return b.NewRecord()
}

type PortfolioRecordBuilder struct {
	b   *array.RecordBuilder
	mem memory.Allocator
}

func NewPortfolioRecordBuilder(mem memory.Allocator, schema *arrow.Schema) *PortfolioRecordBuilder {
	return &PortfolioRecordBuilder{
		b: array.NewRecordBuilder(mem, schema),
	}
}

func (b *PortfolioRecordBuilder) Append(tenantID, portfolioID, isin string, marketValue, exposurePct float64) {
	b.b.Field(0).(*array.StringBuilder).Append(tenantID)
	b.b.Field(1).(*array.StringBuilder).Append(portfolioID)
	b.b.Field(2).(*array.StringBuilder).Append(isin)
	b.b.Field(3).(*array.Float64Builder).Append(marketValue)
	b.b.Field(4).(*array.Float64Builder).Append(exposurePct)
}

func (b *PortfolioRecordBuilder) NewRecord() arrow.Record {
	return b.b.NewRecord()
}

func (b *PortfolioRecordBuilder) Release() {
	b.b.Release()
}
