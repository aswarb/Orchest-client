package set

type Set[T comparable] struct {
	values map[T]struct{}
}

func (s *Set[T]) Add(item T) {
	s.values[item] = struct{}{}
}

func (s *Set[T]) Contains(item T) {
	s.values[item] = struct{}{}
}
