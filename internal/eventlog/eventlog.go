package eventlog

// LogFile interface captures admissable operations on a log file
type LogFile interface {
	Log(event interface{}) error
	Iterator() (LogIterator, error)
	Close() error
	Size() (int, error)
}

type LogIterator interface {
	Next() interface{}
	Close() error
}

type funcLogIterator struct {
	nextFunc func() interface{}
	closeFunc func() error
}

func (f *funcLogIterator) Next() interface{} {
	return f.nextFunc()
}

func (f *funcLogIterator) Close() error {
	return f.closeFunc()
}