// Package store 收敛文件存储的通用原语：原子写、JSON 整文件读写、JSON 分节读写，
// 并经 Store 接口把它们适配为可注入的类型化存取。
//
// 只依赖标准库，不记录日志；一切失败经返回值或序列产出，降级策略归调用方（能力层纪律）。
// 路径由调用方给定，无环境副作用。
// 只收「已有消费方」的原语，不预埋无消费方的 API。
// 统一契约：所有写操作（WriteFileAtomic / SaveJSON / SaveSection）自动递归创建父目录；
// 所有错误均包装为哨兵错误（见 errors.go），调用方用 errors.Is 或 Is* 助手判别失败形态。
// 各文件按主题内聚（atomic/json/section/store/errors），主题之间无依赖。
package store
