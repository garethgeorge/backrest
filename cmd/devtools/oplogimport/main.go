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
	"github.com/garethgeorge/backrest/internal/oplog/bboltstore"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	outpath = flag.String("import-oplog-path", "", "path to import the oplog from compressed textproto e.g. .textproto.gz")
)

func main() {
	flag.Parse()

	if *outpath == "" {
		flag.Usage()
		return
	}

	// create a reader from the file
	f, err := os.Open(*outpath)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	cr, err := gzip.NewReader(f)
	if err != nil {
		log.Fatalf("error creating gzip reader: %v", err)
	}
	defer cr.Close()

	// read into a buffer
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(cr); err != nil {
		log.Printf("error reading from gzip reader: %v", err)
	}

	log.Printf("importing operations from %q", *outpath)

	output := &v1.OperationList{}
	if err := prototext.Unmarshal(buf.Bytes(), output); err != nil {
		log.Fatalf("error unmarshalling operations: %v", err)
	}

	zap.S().Infof("importing %d operations", len(output.Operations))

	oplogFile := path.Join(env.DataDir(), "oplog.boltdb")
	opstore, err := bboltstore.NewBboltStore(oplogFile)
	if err != nil {
		log.Fatalf("error creating oplog : %v", err)
	}
	defer opstore.Close()

	for _, op := range output.Operations {
		if err := opstore.Add(op); err != nil {
			log.Printf("error adding operation to oplog: %v", err)
		}
	}
}
