package gomib

// dedup returns items with duplicates removed, preserving first-occurrence order.
func dedup[T comparable](items []T) []T {
	seen := make(map[T]struct{}, len(items))
	var result []T
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
