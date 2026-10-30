// Package di 提供一个极简的依赖注入容器。
// 生命周期：Register 注册构造函数 → Freeze 一次性全量构建 → Get / All / AllOf 只读。
//
// # 错误边界
//
// 一切真实错误都发生在 Freeze 及以前，以 error 返回：
//   - Register：非法构造函数签名、重复注册、注册裸切片；
//   - Freeze：缺依赖、依赖环、构造函数返回 error、构造出 nil 接口服务。
//
// 冻结后的读操作只有一种 panic 场景：容器未正常 Freeze 就读（代码 bug）。
// 其余读取都是正常逻辑：Get 未命中返回 (零值, false)；
// All / AllOf 只是遍历，结果可能为空。
//
// # 构造函数
//
// 构造函数就是普通 Go 函数，不感知 di：
//
//	func(deps...) T           普通构造
//	func(deps...) (T, error)  可失败构造，error 在 Freeze 时统一上抛
//
// 每个参数按类型匹配一个已注册服务（即依赖），第一个返回值即提供的服务类型。
// 同一类型只能注册一次；Freeze 时每个类型恰好构造一次并全局共享（单例）。
// 构造函数返回 nil 接口服务视为装配错误（避免读侧拿到不明 nil）。
//
// 类型即身份：解析与判断一律按注册类型（第一个返回值），而非实例的动态类型。
// 例如 func() io.Writer 返回 *os.File，只能按 io.Writer 取，按 *os.File 取不到。
//
// # 收集注入
//
// 参数里的裸切片 []T 是保留语法：装配为"注册类型可赋值给 T 的全部服务"，按类型名排序；
// 用于把多个实现汇聚给一个消费者，如：
//
//	func NewServer(cfg *Config, handlers []Handler) *Server
//
// 因此裸切片不能注册为服务；
// 要提供集合值请改用命名切片类型（type Foos []Foo），它是普通服务，不参与收集注入。
//
// # 读取
//
// Get[T] 按注册类型精确解析，未命中返回 false；
// AllOf[T] 遍历可赋值给 T 的全部服务（接口即全部实现，具体类型即精确匹配）；
// All 遍历全部 (注册类型, 实例)。
// 三者产出均按类型名稳定排序；Freeze 后容器不可变，并发读取安全。
//
// # 用法示例
//
//	c, err := di.RegisterAll(
//		di.Value(cfg), // 现成值直接注册
//		NewDB,         // func(cfg *Config) (*DB, error)
//		NewServer,     // func(cfg *Config, handlers []Handler) *Server
//	)
//	if err != nil {
//		return err
//	}
//	if db, ok := di.Get[*DB](c); ok {
//		// ...
//	}
//	for h := range di.AllOf[Handler](c) {
//		// ...
//	}
package di
