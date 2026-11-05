package store

// Store 是一份类型化数据的存取接口，供依赖注入消费；
// 失败形态经哨兵错误（见 errors.go）上抛，降级策略归调用方。
type Store[T any] interface {
	Save(v T) error
	Load() (T, error)
}

// store 用闭包把既有原语适配成 Store。
type store[T any] struct {
	save func(v T) error
	load func() (T, error)
}

func (s store[T]) Save(v T) error   { return s.save(v) }
func (s store[T]) Load() (T, error) { return s.load() }

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
