// Package jsonl 提供 JSONL（JSON Lines，每行一条 JSON 记录）文件的行式追加写原语。
//
// 只依赖标准库，不记录日志；路径由调用方给定，写操作自动递归创建父目录。
// O_APPEND 小块写由 POSIX 保证原子，无需跨进程锁（单写者追加语义）；
// 追加前检测末尾残缺行并自动隔断，坏行不向后续写入传播。
package jsonl
