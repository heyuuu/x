// Package store 收敛文件存储的通用原语：原子写、JSON 整文件读写、JSONL 行式读写。
//
// 只依赖标准库 + slog，路径由调用方给定，无环境副作用（能力层纪律）。
// 只收「已有消费方」的原语，不预埋无消费方的 API。
// 统一契约：所有写操作（WriteFileAtomic / SaveJSON / AppendJsonl）自动递归创建父目录。
// 各文件按主题内聚（atomic/json/jsonl），主题之间无依赖。
package store
