package batch

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// BatchConfig 定义服务配置
type BatchConfig struct {
	Concurrency int        // 最大并发数
	RateLimit   rate.Limit // 每秒限制多少次 (RPS)
	Burst       int        // 突发桶大小
	IgnoreError bool       // 是否忽略错误（true: 遇到错误继续；false: 遇到错误立即终止）
}

// BatchExecutor 通用批处理执行器
type BatchExecutor[T any] struct {
	config BatchConfig
}

// NewBatchExecutor 创建一个新的执行器
func NewBatchExecutor[T any](opts ...func(*BatchConfig)) *BatchExecutor[T] {
	// 默认配置
	cfg := BatchConfig{
		Concurrency: 10,
		RateLimit:   rate.Inf, // 默认不限流
		Burst:       1,
		IgnoreError: false,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &BatchExecutor[T]{config: cfg}
}

// WithConcurrency 配置函数
func WithConcurrency(n int) func(*BatchConfig) {
	return func(c *BatchConfig) {
		if n > 0 {
			c.Concurrency = n
		}
	}
}

func WithRateLimit(rps int) func(*BatchConfig) {
	return func(c *BatchConfig) {
		if rps > 0 {
			c.RateLimit = rate.Limit(rps)
		}
	}
}

// WithBurst 配置限流桶大小，需与 WithRateLimit 配合使用
func WithBurst(n int) func(*BatchConfig) {
	return func(c *BatchConfig) {
		if n > 0 {
			c.Burst = n
		}
	}
}

func WithIgnoreError(ignore bool) func(*BatchConfig) {
	return func(c *BatchConfig) {
		c.IgnoreError = ignore
	}
}

// Execute 执行批处理
// ctx: 上下文，支持超时控制
// items: 数据切片
// handler: 处理逻辑
func (b *BatchExecutor[T]) Execute(ctx context.Context, items []T, handler func(context.Context, T) error) error {
	if len(items) == 0 {
		return nil
	}

	// 1. 初始化 errgroup 和 limiter
	g, grpCtx := errgroup.WithContext(ctx)
	g.SetLimit(b.config.Concurrency)

	var limiter *rate.Limiter
	if b.config.RateLimit != rate.Inf {
		limiter = rate.NewLimiter(b.config.RateLimit, b.config.Burst)
	}

	// 2. 错误收集器 (如果 IgnoreError 为 true)
	var (
		errLock sync.Mutex
		errs    []error
	)

	for i, val := range items {
		// Go 1.22+ 之前需要这一步，1.22 后可省略，为兼容性保留
		v := val
		index := i

		// 3. 检查 Context 是否已取消（Fail Fast）
		if grpCtx.Err() != nil {
			break
		}

		// 4. 限流等待
		if limiter != nil {
			if err := limiter.Wait(grpCtx); err != nil {
				return fmt.Errorf("rate limiter error at index %d: %w", index, err)
			}
		}

		g.Go(func() error {
			// 5. Panic 恢复 (保护主程)；生产环境建议接入 logs.Logger
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("panic recovered processing item %d: %v\n", index, r)
				}
			}()

			// 执行业务逻辑
			err := handler(grpCtx, v)

			if err != nil {
				if b.config.IgnoreError {
					// 模式 A: 记录错误，不返回错误给 errgroup，让其他任务继续
					errLock.Lock()
					errs = append(errs, fmt.Errorf("item %d failed: %w", index, err))
					errLock.Unlock()
					return nil
				}
				// 模式 B: 返回错误，errgroup 会取消 context，停止后续任务
				return err
			}
			return nil
		})
	}

	// 等待所有任务完成
	if err := g.Wait(); err != nil {
		return err // 返回第一个遇到的错误 (Fail Fast)
	}

	if b.config.IgnoreError && len(errs) > 0 {
		return fmt.Errorf("batch finished with %d error(s), first: %w", len(errs), errs[0])
	}

	return nil
}
