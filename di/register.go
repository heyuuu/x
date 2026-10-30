package di

// RegisterAll 便捷入口：注册全部构造函数并冻结容器，任一步失败返回 nil 容器。
func RegisterAll(ctors ...any) (*Container, error) {
	c := New()
	for _, ctor := range ctors {
		if err := c.Register(ctor); err != nil {
			return nil, err
		}
	}
	if err := c.Freeze(); err != nil {
		return nil, err
	}
	return c, nil
}

// Value 把现成值包装成构造函数，免去手写闭包：c.Register(di.Value(cfg))。
// 值的声明类型即注册的服务类型。
func Value[T any](instance T) func() T {
	return func() T {
		return instance
	}
}
