package store

// Store 是一份类型化数据的存取接口，供依赖注入消费；
// 失败形态经哨兵错误（见 errors.go）上抛，降级策略归调用方。
type Store[T any] interface {
	Save(v T) error
	Load() (T, error)
	LoadOrZero() T
}

// store 用闭包把既有原语适配成 Store。
type store[T any] struct {
	save func(v T) error
	load func() (T, error)
}

func (s store[T]) Save(v T) error   { return s.save(v) }
func (s store[T]) Load() (T, error) { return s.load() }
func (s store[T]) LoadOrZero() T {
	v, err := s.load()
	if err != nil {
		var zero T
		return zero
	}
	return v
}

// JSONStore 把整文件 JSON 读写（SaveJSON / LoadJSON）适配为 Store。
func JSONStore[T any](path string) Store[T] {
	return store[T]{
		save: func(v T) error { return SaveJSON(path, v) },
		load: func() (T, error) { return LoadJSON[T](path) },
	}
}

// SectionStore 把 JSON 配置分节读写（SaveSection / LoadSection）适配为 Store。
func SectionStore[T any](file string, section string) Store[T] {
	return store[T]{
		save: func(v T) error { return SaveSection(file, section, v) },
		load: func() (T, error) { return LoadSection[T](file, section) },
	}
}

// WrapperStore 把内部 Store[I] 适配为 Store[O]：Load 经 wrap 转出，Save 经 unwrap 转入。
func WrapperStore[I, O any](inner Store[I], wrap func(I) (O, error), unwrap func(O) (I, error)) Store[O] {
	if wrap == nil {
		panic("store: wrap cannot be nil")
	}
	if unwrap == nil {
		panic("store: unwrap cannot be nil")
	}

	save := func(out O) error {
		in, err := unwrap(out)
		if err != nil {
			return err
		}
		return inner.Save(in)
	}

	load := func() (O, error) {
		in, err := inner.Load()
		if err != nil {
			var zero O
			return zero, err
		}
		return wrap(in)
	}

	return store[O]{save: save, load: load}
}
