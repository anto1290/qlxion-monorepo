package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// Server represents the gRPC server
type Server struct {
	grpcServer *grpc.Server
	port       int
}

// NewServer creates a new gRPC server
func NewServer(port int) *Server {
	grpcServer := grpc.NewServer()
	
	return &Server{
		grpcServer: grpcServer,
		port:       port,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	log.Info().
		Str("addr", addr).
		Msg("gRPC server starting")

	return s.grpcServer.Serve(lis)
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	log.Info().Msg("Stopping gRPC server")
	s.grpcServer.GracefulStop()
}

// RegisterServices registers all gRPC services
func (s *Server) RegisterServices() {
	// TODO: Register gRPC services here
	// Example:
	// pb.RegisterAuthServiceServer(s.grpcServer, &authServiceServer{})
}

// AuthServiceServer implements the AuthService gRPC interface
type AuthServiceServer struct {
	// TODO: Add usecase dependencies
}

// ValidateToken validates a JWT token
func (s *AuthServiceServer) ValidateToken(ctx context.Context, req *struct{ Token string }) (*struct{ Valid bool }, error) {
	// TODO: Implement token validation
	return &struct{ Valid bool }{Valid: true}, nil
}
