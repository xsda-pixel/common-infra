package stream

import (
	"context"
	"fmt"
	"sync"
)

// Worker 流式工作池
type Worker[T any] struct {
	concurrency int
	handler     func(context.Context, T) error
}

// NewStreamWorker 创建工作池
// concurrency: 并发工人数
// handler: 每个数据的处理逻辑
func NewStreamWorker[T any](concurrency int, handler func(context.Context, T) error) *Worker[T] {
	if concurrency <= 0 {
		concurrency = 1
	}
	return &Worker[T]{
		concurrency: concurrency,
		handler:     handler,
	}
}

// Start 开始工作（阻塞模式，直到 channel 关闭且任务全部做完）。
// 注意：handler 返回的 error 仅做内部处理，不会向调用方返回；如需收集错误请自行在 handler 内处理。
// ctx: 用于控制整体退出的上下文
// ch: 数据输入通道 (接收数据的地方)
func (s *Worker[T]) Start(ctx context.Context, ch <-chan T) {
	var wg sync.WaitGroup

	// 启动 N 个工人 (Goroutine)
	for i := 0; i < s.concurrency; i++ {
		wg.Add(1)
		workerID := i // 记录这是第几个工人
		go func() {
			defer wg.Done()

			// 核心逻辑：循环从 channel 拿数据
			for {
				select {
				case <-ctx.Done(): // 上下文取消，停止工作
					return
				case val, ok := <-ch:
					if !ok {
						// 通道关闭了，而且数据取完了，下班
						return
					}

					// 开始干活
					// 这里的 Panic 保护很有必要，防止一个数据搞挂整个池子
					func() {
						defer func() {
							if r := recover(); r != nil {
								fmt.Printf("[Worker %d] Panic: %v\n", workerID, r)
							}
						}()

						// 执行用户逻辑
						_ = s.handler(ctx, val)
					}()
				}
			}
		}()
	}

	// 等待所有工人下班
	wg.Wait()
}
