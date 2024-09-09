#!/usr/bin/env bash

BASEDIR=$(dirname "$0")
RUNDIR=$1

if [ -z "$RUNDIR" ]; then
  echo "Usage: $0 <run-dir>"
  exit 1
fi

go run $BASEDIR/../../cmd/backrest --config-file=$RUNDIR/config.json --data-dir=$RUNDIR

