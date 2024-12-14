package testutil

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"sync/atomic"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/proto"
)

var nextRandomOperationTimeMillis atomic.Int64

func OperationsWithDefaults(op *v1.Operation, ops []*v1.Operation) []*v1.Operation {
	var newOps []*v1.Operation
	for _, o := range ops {
		copy := proto.Clone(o).(*v1.Operation)
		proto.Merge(copy, op)
		newOps = append(newOps, copy)
	}

	return newOps
}

func RandomOperation() *v1.Operation {
	randomPlanID := "plan" + randomString(5)
	randomRepoID := "repo" + randomString(5)
	randomInstanceID := "instance" + randomString(5)

	return &v1.Operation{
		UnixTimeStartMs: nextRandomOperationTimeMillis.Add(1000),
		PlanId:          randomPlanID,
		RepoId:          randomRepoID,
		InstanceId:      randomInstanceID,
		Op:              &v1.Operation_OperationBackup{},
		FlowId:          randomInt(),
		OriginalId:      randomInt(),
		OriginalFlowId:  randomInt(),
		Modno:           randomInt(),
		Status:          v1.OperationStatus_STATUS_INPROGRESS,
	}
}

func randomString(length int) string {
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(randomBytes)
}

func randomInt() int64 {
	randBytes := make([]byte, 8)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic(err)
	}
	return int64(binary.LittleEndian.Uint64(randBytes) & 0x7FFFFFFFFFFFFFFF)
}
