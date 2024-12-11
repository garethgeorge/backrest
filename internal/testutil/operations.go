package testutil

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/proto"
)

func OperationsWithDefaults(op *v1.Operation, ops []*v1.Operation) []*v1.Operation {
	var newOps []*v1.Operation
	for _, o := range ops {
		copy := proto.Clone(o).(*v1.Operation)
		proto.Merge(copy, op)
		newOps = append(newOps, copy)
	}

	return newOps
}
