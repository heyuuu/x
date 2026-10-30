// di 包的单元测试与性能测试

package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- 测试用服务 ----------

type testConfig struct{ DSN string }

type testDB struct{ cfg *testConfig }

type testRepo struct{ db *testDB }

type testService struct {
	repo *testRepo
	cfg  *testConfig
}

// kvStore 用于验证"注册接口类型、构造函数返回具体实现"的场景
type kvStore interface {
	Get(key string) string
}

type memStore struct{ data map[string]string }

func (m *memStore) Get(key string) string { return m.data[key] }

type kvUser struct{ store kvStore }

// fileStore 恰好也实现 kvStore，用于验证 AllOf 按接口收集时
// "注册为具体类型但实现了目标接口" 的服务也会被收进来
type fileStore struct{ path string }

func (f *fileStore) Get(key string) string { return f.path + "/" + key }

// groupUser 消费 []kvStore 收集注入，验证裸切片参数的收集语义
type groupUser struct{ stores []kvStore }

func newGroupUser(stores []kvStore) *groupUser { return &groupUser{stores: stores} }

// storeList 命名切片：验证命名切片类型是普通服务，不参与收集注入
type storeList []kvStore

// assertPanicContains 断言 fn 触发 panic 且消息包含 want
func assertPanicContains(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			assert.Fail(t, fmt.Sprintf("期望 panic 包含 %q, 实际未 panic", want))
			return
		}
		assert.Contains(t, fmt.Sprint(r), want)
	}()
	fn()
}

// mustGet 断言服务已注册并返回实例
func mustGet[T any](t *testing.T, c *Container) T {
	t.Helper()
	v, ok := Get[T](c)
	require.True(t, ok, "Get 应命中已注册服务")
	return v
}

// graphCalls 记录 registerGraph 中各构造函数的调用次数
type graphCalls struct{ config, db, repo, service int }

// registerGraph 注册一条 config <- db <- repo <- service 的依赖链
func registerGraph(t *testing.T, c *Container) *graphCalls {
	t.Helper()
	var n graphCalls
	require.NoError(t, c.Register(func() *testConfig { n.config++; return &testConfig{DSN: "sqlite:memory"} }))
	require.NoError(t, c.Register(func(cfg *testConfig) *testDB { n.db++; return &testDB{cfg: cfg} }))
	require.NoError(t, c.Register(func(db *testDB) *testRepo { n.repo++; return &testRepo{db: db} }))
	require.NoError(t, c.Register(func(repo *testRepo, cfg *testConfig) *testService {
		n.service++
		return &testService{repo: repo, cfg: cfg}
	}))
	return &n
}

// ---------- Register ----------

func TestRegisterValid(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func() *testConfig { return &testConfig{} }))
	require.NoError(t, c.Register(func(cfg *testConfig) (*testDB, error) { return &testDB{cfg: cfg}, nil }))
}

func TestRegisterInvalid(t *testing.T) {
	cases := []struct {
		name string
		ctor any
		want string
	}{
		{"非函数", 42, "需要构造函数"},
		{"nil", nil, "需要构造函数"},
		{"变参函数", func(xs ...int) int { return 0 }, "变参"},
		{"无返回值", func() {}, "必须是 func(deps...) T 或 func(deps...) (T, error)"},
		{"仅返回 error", func() error { return nil }, "缺少提供类型"},
		{"第一个返回值是 error", func() (error, error) { return nil, nil }, "缺少提供类型"},
		{"第二个返回值不是 error", func() (int, int) { return 0, 0 }, "第二个返回值必须是 error"},
		{"返回值过多", func() (int, error, string) { return 0, nil, "" }, "必须是 func(deps...) T 或 func(deps...) (T, error)"},
		{"返回裸切片", func() []kvStore { return nil }, "不能注册裸切片"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := New()
			require.ErrorContains(t, c.Register(tc.ctor), tc.want)
		})
	}
}

func TestRegisterDuplicate(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func() *testConfig { return &testConfig{} }))
	require.ErrorContains(t, c.Register(func() *testConfig { return &testConfig{} }), "已注册")
}

func TestRegisterAfterFreeze(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func() *testConfig { return &testConfig{} }))
	require.NoError(t, c.Freeze())
	require.ErrorContains(t, c.Register(func() *testDB { return &testDB{} }), "已冻结")
}

// ---------- Freeze ----------

func TestFreezeBuildsAndWires(t *testing.T) {
	c := New()
	n := registerGraph(t, c)
	require.NoError(t, c.Freeze())

	svc := mustGet[*testService](t, c)

	// 只 Get 了 *testService，其余服务也应已在 Freeze 时全量构建
	assert.Equal(t, graphCalls{config: 1, db: 1, repo: 1, service: 1}, *n,
		"Freeze 应全量构建且各构造函数恰好调用一次")

	cfg := mustGet[*testConfig](t, c)
	repo := mustGet[*testRepo](t, c)
	assert.Same(t, repo, svc.repo, "Service.repo 与容器中的 *testRepo 应为同一实例")
	assert.Same(t, cfg, svc.cfg, "Service.cfg 与容器中的 *testConfig 应为同一实例")
	assert.Equal(t, "sqlite:memory", svc.cfg.DSN)
}

func TestFreezeSharedDependencySingleton(t *testing.T) {
	type shared struct{ id int }
	type alpha struct{ s *shared }
	type beta struct{ s *shared }

	c := New()
	var calls int
	require.NoError(t, c.Register(func() *shared { calls++; return &shared{id: 7} }))
	require.NoError(t, c.Register(func(s *shared) *alpha { return &alpha{s: s} }))
	require.NoError(t, c.Register(func(s *shared) *beta { return &beta{s: s} }))
	require.NoError(t, c.Freeze())

	a := mustGet[*alpha](t, c)
	b := mustGet[*beta](t, c)
	assert.Equal(t, 1, calls, "共享依赖应只构造一次")
	assert.Same(t, a.s, b.s, "两个消费方应持有同一实例")
}

func TestFreezeCtorError(t *testing.T) {
	errBoom := errors.New("boom")
	c := New()
	require.NoError(t, c.Register(func() (*testDB, error) { return nil, errBoom }))

	err := c.Freeze()
	require.ErrorContains(t, err, "testDB")
	assert.ErrorIs(t, err, errBoom)
}

func TestFreezeNilService(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func() kvStore { return nil }))

	require.ErrorContains(t, c.Freeze(), "nil 服务")
}

func TestFreezeMissingDependency(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func(db *testDB) *testRepo { return &testRepo{db: db} }))

	require.ErrorContains(t, c.Freeze(), "*di.testRepo 依赖 *di.testDB")
}

func TestFreezeCycle(t *testing.T) {
	type nodeA struct{}
	type nodeB struct{}

	c := New()
	require.NoError(t, c.Register(func(b *nodeB) *nodeA { return &nodeA{} }))
	require.NoError(t, c.Register(func(a *nodeA) *nodeB { return &nodeB{} }))

	require.ErrorContains(t, c.Freeze(), "*di.nodeA -> *di.nodeB -> *di.nodeA")
}

func TestFreezeSelfCycle(t *testing.T) {
	type node struct{}

	c := New()
	require.NoError(t, c.Register(func(n *node) *node { return n }))

	require.ErrorContains(t, c.Freeze(), "*di.node -> *di.node")
}

func TestFreezeFailureLeavesUnfrozen(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func(db *testDB) *testRepo { return &testRepo{db: db} }))

	require.Error(t, c.Freeze())
	assert.False(t, c.frozen, "Freeze 失败后容器不应进入冻结状态")

	assertPanicContains(t, "尚未冻结", func() { Get[*testRepo](c) })

	// 补上缺失的依赖后重新 Freeze，失败时写入的部分实例不应泄漏
	require.NoError(t, c.Register(func() *testDB { return &testDB{} }))
	require.NoError(t, c.Freeze())

	_, ok := Get[*testRepo](c)
	require.True(t, ok, "冻结成功后可正常读取")
}

func TestFreezeIdempotent(t *testing.T) {
	c := New()
	n := registerGraph(t, c)
	require.NoError(t, c.Freeze())
	require.NoError(t, c.Freeze()) // 重复调用是 no-op

	assert.Equal(t, graphCalls{config: 1, db: 1, repo: 1, service: 1}, *n, "第二次 Freeze 不应重复构造")
}

func TestFreezeEmptyContainer(t *testing.T) {
	require.NoError(t, New().Freeze())
}

// ---------- RegisterAll (helpers.go) ----------

func TestRegisterAll(t *testing.T) {
	c, err := RegisterAll(
		func() *testConfig { return &testConfig{DSN: "sqlite:memory"} },
		func(cfg *testConfig) *testDB { return &testDB{cfg: cfg} },
	)
	require.NoError(t, err)

	assert.Equal(t, "sqlite:memory", mustGet[*testDB](t, c).cfg.DSN)
}

func TestRegisterAllRegisterError(t *testing.T) {
	c, err := RegisterAll(
		func() *testConfig { return &testConfig{} },
		42, // 非函数
	)
	require.ErrorContains(t, err, "需要构造函数")
	assert.Nil(t, c, "失败时应返回 nil 容器")
}

func TestRegisterAllFreezeError(t *testing.T) {
	c, err := RegisterAll(func(db *testDB) *testRepo { return &testRepo{db: db} })
	require.ErrorContains(t, err, "未注册")
	assert.Nil(t, c, "失败时应返回 nil 容器")
}

func TestValue(t *testing.T) {
	cfg := &testConfig{DSN: "sqlite:memory"}
	var store kvStore = &memStore{data: map[string]string{"k": "v"}}

	c, err := RegisterAll(Value(cfg), Value(store))
	require.NoError(t, err)

	assert.Same(t, cfg, mustGet[*testConfig](t, c), "Value 包装的指针应原样入容器")
	assert.Equal(t, "v", mustGet[kvStore](t, c).Get("k"), "按接口类型声明的 Value 应以接口类型注册")
}

// ---------- Get ----------

func TestGetBeforeFreeze(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func() *testConfig { return &testConfig{} }))

	assertPanicContains(t, "尚未冻结", func() { Get[*testConfig](c) })
}

func TestGetUnregistered(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func() *testConfig { return &testConfig{} }))
	require.NoError(t, c.Freeze())

	// 未注册是正常逻辑（可选服务查询），返回 false 而非 panic
	v, ok := Get[*testDB](c)
	assert.False(t, ok, "未注册应返回 false")
	assert.Nil(t, v, "未命中时返回零值")
}

func TestInterfaceService(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func() kvStore {
		return &memStore{data: map[string]string{"k": "v"}}
	}))
	require.NoError(t, c.Register(func(s kvStore) *kvUser { return &kvUser{store: s} }))
	require.NoError(t, c.Freeze())

	u := mustGet[*kvUser](t, c)
	assert.Equal(t, "v", u.store.Get("k"), "接口依赖注入的实例应可用")

	// 按接口类型直接 Get，取回的是注册的实现
	store := mustGet[kvStore](t, c)
	assert.Equal(t, "v", store.Get("k"), "按接口 Get 应取回注册的实现")
}

func TestConcurrentGetAfterFreeze(t *testing.T) {
	c := New()
	n := registerGraph(t, c)
	require.NoError(t, c.Freeze())

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				_, _ = Get[*testService](c) // 冻结后读取不会 panic
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, graphCalls{config: 1, db: 1, repo: 1, service: 1}, *n, "并发 Get 不应触发重复构造")
}

// ---------- All / AllOf (read.go) ----------

func TestAll(t *testing.T) {
	cfg := &testConfig{DSN: "sqlite:memory"}
	db := &testDB{}
	c, err := RegisterAll(Value(cfg), Value(db))
	require.NoError(t, err)

	all := All(c)

	var order []string
	got := map[reflect.Type]any{}
	for typ, inst := range all {
		order = append(order, typ.String())
		got[typ] = inst
	}
	assert.Equal(t, []string{"*di.testConfig", "*di.testDB"}, order, "应按类型名稳定排序")
	assert.Same(t, cfg, got[reflect.TypeFor[*testConfig]()])
	assert.Same(t, db, got[reflect.TypeFor[*testDB]()])

	for range All(c) { // 提前 break 不应出错
		break
	}
}

func TestAllBeforeFreeze(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(Value(&testConfig{})))

	assertPanicContains(t, "尚未冻结", func() { All(c) })
}

func TestAllOf(t *testing.T) {
	c, err := RegisterAll(
		func() kvStore { return &memStore{data: map[string]string{"k": "v"}} }, // 注册为接口
		func() *fileStore { return &fileStore{path: "tmp"} },                   // 注册为具体类型，恰好也实现 kvStore
		Value(&testConfig{}), // 与 kvStore 无关
	)
	require.NoError(t, err)

	// 按接口收集：接口注册与具体类型注册（只要注册类型实现 kvStore）都应产出
	var got []kvStore
	for s := range AllOf[kvStore](c) {
		got = append(got, s)
	}
	assert.Len(t, got, 2)

	// T 为具体类型时退化为精确匹配
	var fs []*fileStore
	for s := range AllOf[*fileStore](c) {
		fs = append(fs, s)
	}
	assert.Len(t, fs, 1)

	// 按注册类型而非实例动态类型判断：memStore 实例的动态类型是 *memStore，
	// 但其注册类型是 kvStore，因此 AllOf[*memStore] 不应产出它
	count := 0
	for range AllOf[*memStore](c) {
		count++
	}
	assert.Zero(t, count)
}

func TestAllOfBeforeFreeze(t *testing.T) {
	c := New()
	require.NoError(t, c.Register(func() kvStore { return &memStore{} }))

	assertPanicContains(t, "尚未冻结", func() { AllOf[kvStore](c) })
}

func TestAllOfSorted(t *testing.T) {
	c, err := RegisterAll(
		func() kvStore { return &memStore{} },
		func() *fileStore { return &fileStore{} },
	)
	require.NoError(t, err)

	var got []string
	for s := range AllOf[kvStore](c) {
		got = append(got, fmt.Sprintf("%T", s))
	}
	assert.Equal(t, []string{"*di.fileStore", "*di.memStore"}, got, "AllOf 应按类型名稳定排序")
}

// ---------- 收集注入 ([]T 参数) ----------

func TestGroup(t *testing.T) {
	c, err := RegisterAll(
		func() kvStore { return &memStore{data: map[string]string{"k": "v"}} },
		func() *fileStore { return &fileStore{path: "tmp"} },
		Value(&testConfig{}), // 与 kvStore 无关，不应被收集
		newGroupUser,
	)
	require.NoError(t, err)

	u := mustGet[*groupUser](t, c)
	require.Len(t, u.stores, 2, "[]kvStore 参数应收齐注册类型可赋值给 kvStore 的服务")
	// 按类型名排序：*di.fileStore 排在 di.kvStore 之前
	assert.Equal(t, "tmp/k", u.stores[0].Get("k"), "收集成员应是容器内同一实例")
	assert.Equal(t, "v", u.stores[1].Get("k"), "收集成员应是容器内同一实例")
}

func TestGroupEmpty(t *testing.T) {
	c, err := RegisterAll(newGroupUser)
	require.NoError(t, err)

	assert.Empty(t, mustGet[*groupUser](t, c).stores, "无候选服务时应注入空切片")
}

func TestGroupCycle(t *testing.T) {
	c := New()
	// groupUser 依赖 []kvStore，候选实现又依赖 groupUser → 应报依赖环
	require.NoError(t, c.Register(func(u *groupUser) *memStore { return &memStore{} }))
	require.NoError(t, c.Register(newGroupUser))

	require.ErrorContains(t, c.Freeze(), "依赖环")
}

func TestNamedSliceService(t *testing.T) {
	list := storeList{&memStore{data: map[string]string{"k": "v"}}}
	c, err := RegisterAll(
		Value(list),
		func(l storeList) *kvUser { return &kvUser{store: l[0]} },
	)
	require.NoError(t, err)

	u := mustGet[*kvUser](t, c)
	assert.Equal(t, "v", u.store.Get("k"), "命名切片类型应作为普通服务解析，而非收集注入")
}

// ---------- 性能测试 ----------

// 10 层依赖链 b1 <- b2 <- ... <- b10，用于测 Freeze/Get 开销
type b1 struct{ v int }
type b2 struct{ v int }
type b3 struct{ v int }
type b4 struct{ v int }
type b5 struct{ v int }
type b6 struct{ v int }
type b7 struct{ v int }
type b8 struct{ v int }
type b9 struct{ v int }
type b10 struct{ v int }

func registerBenchChain(c *Container) error {
	ctors := []any{
		func() *b1 { return &b1{v: 1} },
		func(a *b1) *b2 { return &b2{v: a.v + 1} },
		func(a *b2) *b3 { return &b3{v: a.v + 1} },
		func(a *b3) *b4 { return &b4{v: a.v + 1} },
		func(a *b4) *b5 { return &b5{v: a.v + 1} },
		func(a *b5) *b6 { return &b6{v: a.v + 1} },
		func(a *b6) *b7 { return &b7{v: a.v + 1} },
		func(a *b7) *b8 { return &b8{v: a.v + 1} },
		func(a *b8) *b9 { return &b9{v: a.v + 1} },
		func(a *b9) *b10 { return &b10{v: a.v + 1} },
	}
	for _, ctor := range ctors {
		if err := c.Register(ctor); err != nil {
			return err
		}
	}
	return nil
}

func newFrozenChain(b *testing.B) *Container {
	b.Helper()
	c := New()
	if err := registerBenchChain(c); err != nil {
		b.Fatal(err)
	}
	if err := c.Freeze(); err != nil {
		b.Fatal(err)
	}
	return c
}

func BenchmarkRegister(b *testing.B) {
	ctor := func() *testConfig { return &testConfig{DSN: "bench"} }
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		c := New()
		b.StartTimer()
		if err := c.Register(ctor); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFreeze(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		c := New()
		if err := registerBenchChain(c); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := c.Freeze(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	c := newFrozenChain(b)
	b.ReportAllocs()
	var sink *b10
	for b.Loop() {
		sink, _ = Get[*b10](c)
	}
	_ = sink
}

func BenchmarkGetParallel(b *testing.B) {
	c := newFrozenChain(b)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var sink *b10
		for pb.Next() {
			sink, _ = Get[*b10](c)
		}
		_ = sink
	})
}
