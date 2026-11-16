package remotereg

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/typepb"
)

func BenchmarkRemoteResolution(b *testing.B) {
	for _, count := range []int{1, 64} {
		b.Run(fmt.Sprintf("types_%d", count), func(b *testing.B) {
			types := make(map[string]*typepb.Type, count)
			for i := range count {
				name := fmt.Sprintf("bench.Message%d", i)
				msg := &typepb.Type{Name: name, SourceContext: &sourcecontextpb.SourceContext{FileName: "bench.proto"}}
				if i+1 < count {
					msg.Fields = []*typepb.Field{{Name: "child", Number: 1, Kind: typepb.Field_TYPE_MESSAGE, Cardinality: typepb.Field_CARDINALITY_OPTIONAL, TypeUrl: fmt.Sprintf("https://example.com/bench.Message%d", i+1)}}
				}

				types["https://example.com/"+name] = msg
			}

			fetcher := TypeFetcherFunc(func(_ context.Context, url string, _ bool) (proto.Message, error) {
				return types[url], nil
			})
			b.ReportAllocs()
			for b.Loop() {
				reg := &Registry{TypeFetcher: fetcher}
				if _, err := reg.FindMessageByURL("https://example.com/bench.Message0"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
