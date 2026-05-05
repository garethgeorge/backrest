package ioutil

import "iter"

const DefaultBatchSize = 512

func Batchify[T any](items []T, batchSize int) iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		for i := 0; i < len(items); i += batchSize {
			end := min(i+batchSize, len(items))
			if !yield(items[i:end]) {
				return
			}
		}
	}
}
