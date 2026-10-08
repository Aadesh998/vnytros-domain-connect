package utils

func ClampPageSize(limit int) int {
	if limit <= 0 {
		return DefaultPageSize
	}
	if limit > MaxPageSize {
		return MaxPageSize
	}
	return limit
}

func BuildPage[T any](items []T, limit int, cursorFn func(T) uint) (data []T, nextCursor uint, hasMore bool) {
	hasMore = len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	if hasMore && len(items) > 0 {
		nextCursor = cursorFn(items[len(items)-1])
	}
	if items == nil {
		items = []T{}
	}
	return items, nextCursor, hasMore
}
