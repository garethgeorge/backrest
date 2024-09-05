package logwriter

import "errors"

var ErrFileNotFound = errors.New("file not found")
var ErrNotFound = errors.New("entry not found")
var ErrBadName = errors.New("bad name")
