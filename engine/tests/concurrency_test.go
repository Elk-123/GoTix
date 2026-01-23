package tests

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Elk-123/gotix/engine" // 引用你的核心库
	"github.com/redis/go-redis/v9"
)

func TestHighConcurrencyBooking(t *testing.T) {
	// 1. 连接本地 Redis (确保你本地运行了 Redis，默认端口 6379)
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 检查 Redis 是否活着
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("⚠️ 跳过测试: 本地没有检测到 Redis 服务，请启动 Docker Redis")
	}

	// 2. 初始化引擎
	svc, err := engine.NewEngine(
		engine.WithRedis(rdb),
		engine.WithLockTTL(5*time.Second),
	)
	if err != nil {
		t.Fatalf("引擎初始化失败: %v", err)
	}

	// 模拟数据
	eventID := "jay_chou_concert"
	seatID := int64(666) // 大家都抢这个 666 号座

	// 清理脏数据，确保测试环境纯净
	rdb.Del(context.Background(), fmt.Sprintf("gotix:%s:seats", eventID))
	rdb.Del(context.Background(), fmt.Sprintf("gotix:%s:lock:%d", eventID, seatID))

	// 3. 模拟 100 个并发枪手
	concurrency := 100
	var successCount int32
	var wg sync.WaitGroup

	t.Logf("🔥 开始压力测试: %d 个用户正在疯抢座位 %d ...", concurrency, seatID)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			user := fmt.Sprintf("user_%d", uid)

			// --- 核心动作：抢座 ---
			ok, _ := svc.LockSeat(context.Background(), eventID, seatID, user)

			if ok {
				atomic.AddInt32(&successCount, 1)
				t.Logf("🎉 恭喜! 用户 [%s] 抢到了座位!", user)
			}
		}(i)
	}

	wg.Wait()

	// 4. 最终审判
	if successCount == 1 {
		t.Log("✅ 测试通过! 系统完美防住了并发冲突，只有 1 人抢到。")
	} else {
		t.Errorf("❌ 测试失败! 竟然有 %d 人抢到了同一个座位 (预期应为 1)", successCount)
	}
}
