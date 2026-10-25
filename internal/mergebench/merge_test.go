package mergebench

import (
	"fmt"
	"io"
	"testing"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/ad3n/protoreflect/v2/protobuilder"
	"github.com/ad3n/protoreflect/v2/protodescs"
	"github.com/ad3n/protoreflect/v2/protomessage"
	"github.com/ad3n/protoreflect/v2/protoprint"
	"github.com/ad3n/protoreflect/v2/protoresolve"
	"github.com/ad3n/protoreflect/v2/protoresolve/remotereg"
	"github.com/ad3n/protoreflect/v2/sourceinfo"
)

func BenchmarkMergeDescriptorPaths(b *testing.B) {
	for _, count := range []int{8, 256} {
		b.Run(fmt.Sprintf("fields_%d", count), func(b *testing.B) {
			fp := &descriptorpb.FileDescriptorProto{Name: new("bench.proto"), Package: new("bench"), Syntax: new("proto2"), MessageType: []*descriptorpb.DescriptorProto{{Name: new("Record")}}}
			for i := range count {
				fp.MessageType[0].Field = append(fp.MessageType[0].Field, &descriptorpb.FieldDescriptorProto{Name: new(fmt.Sprintf("field_%d", i)), Number: new(int32(i + 1)), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(), DefaultValue: new(`payload\000\377`)})
			}

			fd, err := protodesc.NewFile(fp, nil)
			if err != nil {
				b.Fatal(err)
			}

			b.Run("build", func(b *testing.B) {
				for b.Loop() {
					builder, err := protobuilder.FromFile(fd)
					if err != nil {
						b.Fatal(err)
					}

					if _, err = builder.Build(); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("print", func(b *testing.B) {
				p := &protoprint.Printer{}
				for b.Loop() {
					if err := p.PrintProtoFile(fd, io.Discard); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("registry", func(b *testing.B) {
				for b.Loop() {
					reg := &protoresolve.Registry{}
					if _, err := reg.RegisterFileProto(fp); err != nil {
						b.Fatal(err)
					}

					if _, err := reg.FindMessageByName("bench.Record"); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("convert", func(b *testing.B) {
				dc := &remotereg.DescriptorConverter{}
				for b.Loop() {
					_ = dc.DescriptorAsType(fd.Messages().Get(0))
				}
			})
			b.Run("walk", func(b *testing.B) {
				msg := fp.ProtoReflect()
				for b.Loop() {
					protomessage.Walk(msg, func([]any, protoreflect.Message) bool { return true })
				}
			})
			b.Run("sort", func(b *testing.B) {
				files := make([]*descriptorpb.FileDescriptorProto, count)
				for i := range count {
					files[i] = &descriptorpb.FileDescriptorProto{Name: new(fmt.Sprintf("file_%d.proto", i))}
					if i > 0 {
						files[i].Dependency = []string{files[i-1].GetName()}
					}
				}

				input := make([]*descriptorpb.FileDescriptorProto, count)
				for b.Loop() {
					for i := range count {
						input[i] = files[count-i-1]
					}

					if err := protodescs.SortFiles(input); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("sourceinfo", func(b *testing.B) {
				for b.Loop() {
					if _, err := sourceinfo.AddSourceInfoToFile(fd); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
