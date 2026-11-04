package util

// MapSlice 将 in 的每个元素经 fn 映射后收集为新切片；in 为 nil 时返回空切片。
func MapSlice[S any, T any](in []S, fn func(S) T) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		out = append(out, fn(v))
	}
	return out
}

// Keys 返回 map 的全部键，顺序随机；nil map 返回空切片。
func Keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
