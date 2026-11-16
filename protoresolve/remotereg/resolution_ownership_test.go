package remotereg

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/typepb"
)

func TestResolutionPreservesFetcherInputs(t *testing.T) {
	root := &typepb.Type{Name: "bench.Root", Fields: []*typepb.Field{{Name: "child", Number: 1, Kind: typepb.Field_TYPE_MESSAGE, Cardinality: typepb.Field_CARDINALITY_OPTIONAL, TypeUrl: "https://example.com/bench.Child"}}}
	child := &typepb.Type{Name: "bench.Child", Fields: []*typepb.Field{{Name: "parent", Number: 1, Kind: typepb.Field_TYPE_MESSAGE, Cardinality: typepb.Field_CARDINALITY_OPTIONAL, TypeUrl: "https://example.com/bench.Root"}}}
	wantRoot, wantChild := proto.Clone(root), proto.Clone(child)
	reg := &Registry{TypeFetcher: TypeFetcherFunc(func(_ context.Context, url string, _ bool) (proto.Message, error) {
		if url == "https://example.com/bench.Root" {
			return root, nil
		}

		return child, nil
	})}
	first, err := reg.FindMessageByURL("https://example.com/bench.Root")
	if err != nil {
		t.Fatal(err)
	}

	second, err := reg.FindMessageByURL("https://example.com/bench.Root")
	if err != nil || first != second {
		t.Fatalf("cached descriptor changed: %v", err)
	}

	if !proto.Equal(root, wantRoot) || !proto.Equal(child, wantChild) {
		t.Fatal("resolution mutated fetcher-owned messages")
	}
}

func TestDependencyFreeConversionCancellation(t *testing.T) {
	for _, fields := range [][]*typepb.Field{nil, {{Name: "value", Number: 1, Kind: typepb.Field_TYPE_STRING, Cardinality: typepb.Field_CARDINALITY_OPTIONAL}}} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		dc := &DescriptorConverter{}
		_, err := dc.ToMessageDescriptor(ctx, &typepb.Type{Name: "bench.Message", Fields: fields})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled dependency-free conversion returned %v", err)
		}
	}
}
