package grpc

import (
	"context"
	"encoding/hex"
	"github.com/juju/loggo"
	"google.golang.org/grpc"
	"io"
	"lachain-communication-hub/peer"
	"net"

	pb "lachain-communication-hub/grpc/protobuf"
)

var log = loggo.GetLogger("server")

type Server struct {
	pb.UnimplementedCommunicationHubServer
	peer       *peer.Peer
	grpcServer *grpc.Server
	Serve      func()
}

func (s *Server) GetKey(ctx context.Context, in *pb.GetHubIdRequest) (*pb.GetHubIdReply, error) {
	log.Tracef("Received: Get Key Request")
	return &pb.GetHubIdReply{
		Id: s.peer.GetId(),
	}, nil
}

func (s *Server) Init(ctx context.Context, in *pb.InitRequest) (*pb.InitReply, error) {
	log.Tracef("Received: Init Request")
	s.peer.Register(in.GetSignature())
	return &pb.InitReply{
		// TODO: check real result
		Result: true,
	}, nil
}

func (s *Server) Communicate(stream pb.CommunicationHub_CommunicateServer) error {

	log.Debugf("Started new communication server")

	ctx := stream.Context()

	onMsg := func(msg []byte) {
		log.Tracef("On message callback is called")
		select {
		case <-ctx.Done():
			log.Errorf("Unable to send msg via rpc")
			s.peer.SetStreamHandlerFn(peer.GRPCHandlerMock)
			return
		default:
		}

		log.Tracef("Received msg, sending via rpc to client")
		resp := pb.OutboundMessage{Data: msg}
		if err := stream.Send(&resp); err != nil {
			log.Errorf("Unable to send msg via rpc")
			s.peer.SetStreamHandlerFn(peer.GRPCHandlerMock)
		}
	}

	s.peer.SetStreamHandlerFn(onMsg)

	for {

		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// receive data from stream
		req, err := stream.Recv()
		if err == io.EOF {
			// return will close stream from server side
			log.Errorf("exit")
			return err
		}
		if err != nil {
			return err
		}

		log.Tracef("Sending message to peer %s message length %d", hex.EncodeToString(req.PublicKey), len(req.Data))
		s.peer.SendMessageToPeer(hex.EncodeToString(req.PublicKey), req.Data)
	}
}

func runServer(s *grpc.Server, lis net.Listener) {
	log.Infof("GRPC server is listening on %s", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Errorf("failed to serve: %v", err)
	}
}

func New(port string, p *peer.Peer) *Server {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Errorf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	server := &Server{
		peer:       p,
		grpcServer: s,
		Serve: func() {
			runServer(s, lis)
		},
	}
	pb.RegisterCommunicationHubServer(s, server)
	return server
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
