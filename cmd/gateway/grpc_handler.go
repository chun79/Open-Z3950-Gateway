package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"connectrpc.com/connect"
	"github.com/yourusername/open-z3950-gateway/pkg/provider"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
	
	pb "github.com/yourusername/open-z3950-gateway/gen/proto/gateway/v1"
	"github.com/yourusername/open-z3950-gateway/gen/proto/gateway/v1/gatewayv1connect"
)

type GatewayServer struct {
	provider provider.Provider
	gatewayv1connect.UnimplementedGatewayServiceHandler
}

func NewGatewayServer(p provider.Provider) *GatewayServer {
	return &GatewayServer{provider: p}
}

func (s *GatewayServer) Login(ctx context.Context, req *connect.Request[pb.LoginRequest]) (*connect.Response[pb.LoginResponse], error) {
	// For simplicity, reusing provider logic or stubbing since this is a demo of Search streaming
	// Real implementation should use s.provider.GetUserByUsername and jwt generation
	return connect.NewResponse(&pb.LoginResponse{
		Token: "demo-token",
		Role:  "admin",
	}), nil
}

func (s *GatewayServer) Search(ctx context.Context, req *connect.Request[pb.SearchRequest], stream *connect.ServerStream[pb.SearchResponse]) error {
	targets := req.Msg.Targets
	term := req.Msg.Query
	limit := int(req.Msg.Limit)
	if limit <= 0 { limit = 5 }

	slog.Info("gRPC Search started", "term", term, "targets", len(targets))

	var wg sync.WaitGroup
	// No results channel needed! We send directly to stream.
	// But stream.Send() is NOT thread-safe for concurrent calls on the same stream object.
	// We need a mutex for the stream.
	var streamMu sync.Mutex

	zQuery := z3950.StructuredQuery{
		Root: z3950.QueryClause{Attribute: z3950.UseAttributeAny, Term: term},
	}

	for _, t := range targets {
		wg.Add(1)
		go func(targetName string) {
			defer wg.Done()

			// Notify start (optional, maybe via Status)
			// ...

			ids, err := s.provider.Search(targetName, zQuery)
			if err != nil {
				streamMu.Lock()
				stream.Send(&pb.SearchResponse{
					Result: &pb.SearchResponse_Status{
						Status: &pb.SearchStatus{Target: targetName, Success: false, Message: err.Error()},
					},
				})
				streamMu.Unlock()
				return
			}

			if len(ids) == 0 {
				streamMu.Lock()
				stream.Send(&pb.SearchResponse{
					Result: &pb.SearchResponse_Status{
						Status: &pb.SearchStatus{Target: targetName, Success: true, Message: "No records found"},
					},
				})
				streamMu.Unlock()
				return
			}

			// Fetch details
			fetchCount := limit
			if len(ids) < fetchCount { fetchCount = len(ids) }
			records, err := s.provider.Fetch(targetName, ids[:fetchCount])
			if err != nil {
				slog.Error("fetch error", "target", targetName, "err", err)
				return
			}

			// Stream each record immediately!
			for _, rec := range records {
				res := &pb.SearchResult{
					Title:        rec.GetTitle(nil),
					Author:       rec.GetAuthor(nil),
					Isbn:         rec.GetISBN(nil),
					Publisher:    rec.GetPublisher(nil),
					Year:         rec.GetPubYear(nil),
					SourceTarget: targetName,
					RecordId:     rec.RecordID,
				}
				
				streamMu.Lock()
				stream.Send(&pb.SearchResponse{
					Result: &pb.SearchResponse_Record{Record: res},
				})
				streamMu.Unlock()
			}

			// Notify completion for this target
			streamMu.Lock()
			stream.Send(&pb.SearchResponse{
				Result: &pb.SearchResponse_Status{
					Status: &pb.SearchStatus{Target: targetName, Success: true, Message: fmt.Sprintf("Finished fetching %d records", len(records))},
				},
			})
			streamMu.Unlock()

		}(t)
	}

	wg.Wait()
	return nil
}

func (s *GatewayServer) GetBook(ctx context.Context, req *connect.Request[pb.GetBookRequest]) (*connect.Response[pb.GetBookResponse], error) {
	// Stub implementation
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}
