package kvstore

type KvStore interface {
	// Get retrieves the value for the given key.
	Get(key string) ([]byte, error)
	// Set sets the value for the given key.
	Set(key string, value []byte) error
	// ForEach iterates over all key-value pairs with the given prefix.
	ForEach(prefix string, onRow func(key string, value []byte) error) error
}
