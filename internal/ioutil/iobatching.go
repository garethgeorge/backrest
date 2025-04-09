package ioutil

const DefaultBatchSize = 512

func Batchify[T any](items []T, batchSize int) [][]T {
	var batches [][]T
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}
	return batches
}
