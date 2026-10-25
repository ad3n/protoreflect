package mergebench

import (
	"context"
	"net"
	"reflect"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/ad3n/protoreflect/v2/grpcdynamic"
	"github.com/ad3n/protoreflect/v2/grpcreflect"
	grpctesting "github.com/ad3n/protoreflect/v2/internal/testing"
	grpctestprotos "github.com/ad3n/protoreflect/v2/internal/testprotos/grpc"
)

func BenchmarkMergeGRPC(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}

	server := grpc.NewServer()
	grpctestprotos.RegisterTestServiceServer(server, grpctesting.TestService{})
	reflection.Register(server)
	go func() { _ = server.Serve(listener) }()
	b.Cleanup(server.Stop)
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = conn.Close() })
	client := grpcreflect.NewClientAuto(context.Background(), conn)
	b.Cleanup(client.Reset)
	fd, err := client.FileContainingSymbol("grpc.testing.TestService")
	if err != nil {
		b.Fatal(err)
	}

	md := fd.Services().ByName("TestService").Methods().ByName("UnaryCall")
	stub := grpcdynamic.NewStub(conn)
	method := reflect.ValueOf(stub).MethodByName("InvokeRPC")
	if !method.IsValid() {
		method = reflect.ValueOf(stub).MethodByName("InvokeRpc")
	}

	invoke := method.Interface().(func(context.Context, protoreflect.MethodDescriptor, proto.Message, ...grpc.CallOption) (proto.Message, error))
	for _, size := range []struct {
		name  string
		bytes int
	}{{"small", 32}, {"large", 65536}} {
		b.Run("unary_"+size.name, func(b *testing.B) {
			request := &grpctestprotos.SimpleRequest{Payload: &grpctestprotos.Payload{Body: make([]byte, size.bytes)}}
			for b.Loop() {
				if _, err := invoke(context.Background(), md, request); err != nil {
					b.Fatal(err)
				}
			}
		})
	}

	b.Run("reflection_cached", func(b *testing.B) {
		for b.Loop() {
			if _, err := client.FileContainingSymbol("grpc.testing.TestService"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("reflection_cold", func(b *testing.B) {
		for b.Loop() {
			fresh := grpcreflect.NewClientAuto(context.Background(), conn)
			_, err := fresh.FileContainingSymbol("grpc.testing.TestService")
			fresh.Reset()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
