package di

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

var errType = reflect.TypeFor[error]()

// Container 依赖注入容器。
type Container struct {
	services  map[reflect.Type]*service
	instances map[reflect.Type]any
	frozen    bool
}

// service 注册表中的一项，对应一个构造函数
type service struct {
	ctor   reflect.Value
	params []reflect.Type // 构造函数的参数类型，即依赖（按声明顺序）
	hasErr bool
}

// New 创建空容器。
func New() *Container {
	return &Container{services: make(map[reflect.Type]*service)}
}

// Register 注册一个构造函数（签名形态与收集注入约定见包文档）；
// 签名非法、类型重复或已冻结时返回 error。
func (c *Container) Register(ctor any) error {
	if c.frozen {
		return errors.New("di: 容器已冻结, 不能再注册")
	}

	fn := reflect.ValueOf(ctor)
	if fn.Kind() != reflect.Func {
		return fmt.Errorf("di: Register 需要构造函数, 得到 %T", ctor)
	}
	t := fn.Type()
	if t.IsVariadic() {
		return errors.New("di: 不支持变参构造函数")
	}
	var hasErr bool
	switch t.NumOut() {
	case 1:
		if t.Out(0) == errType {
			return errors.New("di: 构造函数缺少提供类型（第一个返回值不能是 error）")
		}
	case 2:
		if t.Out(0) == errType {
			return errors.New("di: 构造函数缺少提供类型（第一个返回值不能是 error）")
		}
		if t.Out(1) != errType {
			return fmt.Errorf("di: 构造函数第二个返回值必须是 error, 得到 %s", t.Out(1))
		}
		hasErr = true
	default:
		return fmt.Errorf("di: 构造函数必须是 func(deps...) T 或 func(deps...) (T, error), 得到 %s", t)
	}

	// 裸切片保留给参数的收集注入（见包文档）
	if out := t.Out(0); out.Kind() == reflect.Slice && out.Name() == "" {
		return fmt.Errorf("di: 不能注册裸切片 %s, []T 保留给参数的收集注入; 集合服务请改用命名类型", out)
	}

	svc := &service{ctor: fn, hasErr: hasErr}
	for i := range t.NumIn() {
		svc.params = append(svc.params, t.In(i))
	}

	typ := t.Out(0)
	if _, exists := c.services[typ]; exists {
		return fmt.Errorf("di: 服务 %s 已注册", typ)
	}
	c.services[typ] = svc
	return nil
}

// Freeze 结束注册阶段并全量构建所有服务，此后容器只读。
// 一切装配错误只在这里以 error 上抛（见包文档"错误边界"）；
// 失败时容器保持未冻结，可补注册后重试；幂等，重复调用是 no-op。
func (c *Container) Freeze() error {
	if c.frozen {
		return nil
	}
	c.instances = make(map[reflect.Type]any, len(c.services))

	path := make([]reflect.Type, 0, len(c.services)) // 当前 DFS 路径
	inPath := make(map[reflect.Type]bool)

	var build func(typ reflect.Type) error
	build = func(typ reflect.Type) error {
		if _, done := c.instances[typ]; done {
			return nil
		}

		// 循环依赖检测
		if inPath[typ] {
			return cycleError(path, typ)
		}

		// 未定义检测
		svc, ok := c.services[typ]
		if !ok {
			return fmt.Errorf("di: %s 依赖 %s, 但后者未注册", path[len(path)-1], typ)
		}

		// 标记为处理中
		inPath[typ] = true
		path = append(path, typ)
		defer func() {
			path = path[:len(path)-1]
			inPath[typ] = false
		}()

		// 构造
		args := make([]reflect.Value, len(svc.params))
		for i, pt := range svc.params {
			// 裸切片是收集注入语法，不是普通依赖查找
			if pt.Kind() == reflect.Slice && pt.Name() == "" {
				group, err := c.collectGroup(pt, build)
				if err != nil {
					return err
				}
				args[i] = group
				continue
			}

			// 常规类型
			if err := build(pt); err != nil {
				return err
			}
			args[i] = reflect.ValueOf(c.instances[pt])
		}
		outs := svc.ctor.Call(args)
		if svc.hasErr && !outs[1].IsNil() {
			return fmt.Errorf("di: 构造 %s 失败: %w", typ, outs[1].Interface().(error))
		}
		// nil 接口服务会让读侧 Get/AllOf 的类型断言 panic 出不明 nil，统一在装配期拦截
		if outs[0].Kind() == reflect.Interface && outs[0].IsNil() {
			return fmt.Errorf("di: 构造 %s 返回了 nil 服务", typ)
		}
		c.instances[typ] = outs[0].Interface()

		return nil
	}

	// 按类型名排序遍历，保证同类错误的信息稳定可复现
	for _, typ := range c.sortedTypes() {
		if err := build(typ); err != nil {
			return err
		}
	}

	c.frozen = true
	return nil
}

// sortedTypes 返回按类型名排序的注册类型，保证遍历顺序与错误信息稳定可复现。
func (c *Container) sortedTypes() []reflect.Type {
	keys := slices.Collect(maps.Keys(c.services))
	slices.SortFunc(keys, func(a, b reflect.Type) int { return cmp.Compare(a.String(), b.String()) })
	return keys
}

// collectGroup 收集注册类型可赋值给 T 的全部服务，装配裸切片参数；
// 只在 Freeze 构建期间可调用。
func (c *Container) collectGroup(groupType reflect.Type, build func(reflect.Type) error) (reflect.Value, error) {
	elem := groupType.Elem()
	out := reflect.MakeSlice(groupType, 0, len(c.services))
	for _, typ := range c.sortedTypes() {
		if !typ.AssignableTo(elem) {
			continue
		}
		if err := build(typ); err != nil {
			return reflect.Value{}, err
		}
		// Freeze 已拦截 nil 接口服务，成员到 elem（接口或同型）的 Convert 必然安全
		out = reflect.Append(out, reflect.ValueOf(c.instances[typ]).Convert(elem))
	}
	return out, nil
}

// cycleError 生成 "A -> B -> A" 形式的环报告。
func cycleError(path []reflect.Type, repeated reflect.Type) error {
	start := slices.Index(path, repeated)
	var b strings.Builder
	for _, typ := range path[start:] {
		b.WriteString(typ.String())
		b.WriteString(" -> ")
	}
	b.WriteString(repeated.String())
	return fmt.Errorf("di: 依赖环: %s", b.String())
}
