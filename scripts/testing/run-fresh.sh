#!/usr/bin/env bash

BASEDIR=$(dirname "$0")
TEMPDIR=$(mktemp -d)

function cleanup {
  echo "Removing temp dir: $TEMPDIR"
  rm -rf $TEMPDIR
}

trap cleanup EXIT

echo "Temp dir: $TEMPDIR"

go run $BASEDIR/../../cmd/backrest --config-file=$TEMPDIR/config.json --data-dir=$TEMPDIR/data
