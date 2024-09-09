package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"log"
	"os"
	"path"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/env"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/bboltstore"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	outpath = flag.String("export-oplog-path", "", "path to export the oplog as a compressed textproto e.g. .textproto.gz")
)

func main() {
	flag.Parse()

	if *outpath == "" {
		flag.Usage()
		return
	}

	oplogFile := path.Join(env.DataDir(), "oplog.boltdb")
	opstore, err := bboltstore.NewBboltStore(oplogFile)
	if err != nil {
		log.Fatalf("error creating oplog : %v", err)
	}
	defer opstore.Close()

	output := &v1.OperationList{}

	l, err := oplog.NewOpLog(opstore)
	if err != nil {
		log.Fatalf("error creating oplog: %v", err)
	}
	l.Query(oplog.Query{}, func(op *v1.Operation) error {
		output.Operations = append(output.Operations, op)
		return nil
	})
	log.Printf("exporting %d operations", len(output.Operations))

	bytes, err := prototext.MarshalOptions{Multiline: true}.Marshal(output)
	if err != nil {
		log.Fatalf("error marshalling operations: %v", err)
	}

	bytes, err = compress(bytes)
	if err != nil {
		log.Fatalf("error compressing operations: %v", err)
	}

	if err := os.WriteFile(*outpath, bytes, 0644); err != nil {
		log.Fatalf("error writing to file: %v", err)
	}
}

func compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	if _, err := zw.Write(data); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
