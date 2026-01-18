# GoTix
战略调整确认：**核心业务 SDK 化 (Core as a Library)** + **SaaS 示范平台 (Reference Implementation)**。

---

### 🛑 第一步：兵棋推演 (Product Strategy & Gap Analysis)

#### 1. 定位与差异化 (Positioning & Differentiation)

*   **竞品对标**：
    *   **Go-Seckill 类项目**：GitHub 上泛滥的秒杀 Demo，通常只是 `Redis.Decr` 的简单封装，缺乏“选座”、“连坐”等复杂状态管理。
    *   **Etcd / Zookeeper**：虽然能做分布式锁，但无法处理“第3排第5座”这种二维坐标的业务语义，且吞吐量不如 Redis。
    *   **商业闭源 (Ticketmaster/大麦)**：极度复杂，耦合了支付、会员等非核心逻辑，无法剥离使用。

*   **痛点狙击**：
    *   开发者想要一个“开箱即用”的高并发选座功能，但不想引入沉重的微服务框架。
    *   **核心痛点**：如何在一个 Go 函数调用中，实现 **Million-level TPS** 的原子选座，且支持 **智能连坐推荐**（位运算查找）。

*   **核心价值 (Value Proposition)**：
    *   **项目名称**：`Galaxy-Engine` (SDK) + `Galaxy-SaaS` (Demo Platform)
    *   **Slogan**：**"The SQLite of Ticketing Engines."** (票务引擎界的 SQLite)
    *   **愿景**：只需一行 `import`，让任何 Go 应用瞬间拥有大麦网级别的抗压选座能力。

#### 2. 受众画像 (User Persona)
*   **SDK 用户**：中高级 Go 后端开发，需要为公司现有的电商/活动系统增加抢票模块。
*   **SaaS 用户**：初创票务公司，直接部署你的 SaaS 平台进行卖票。

#### 3. 技术栈谈判 (Tech Stack Negotiation)

已锁定，强调版本规范：

*   **Core Engine (SDK)**: **Go 1.21+** (必须支持 Generics 泛型，用于处理不同类型的 ID)。
*   **Storage Driver**: **Redis 6.2+** (强依赖 `BITFIELD` 和 `EVAL` Lua 脚本)。
*   **Risk Brain**: **Python 3.10+** (作为 Sidecar 运行，通过 gRPC/HTTP 与 Go 通信)。
*   **SaaS Host**: **Go (Gin/Hertz)** + **MySQL 8.0** (持久化存储)。

---

### 🏗️ 第二步：排兵布阵 (Deep-Dive Architecture)

这是本次架构的精髓：**将核心逻辑“库化”**。

#### 1. 系统全景 (System Topology)

```mermaid
graph TD
    subgraph "External World"
        Web[Web Frontend / App]
        Bot[Scalper Bot]
    end

    subgraph "Galaxy SaaS Platform (The Host)"
        API[API Gateway (Gin)]
        DB[(MySQL - Persistence)]
        
        subgraph "Galaxy-Engine (The SDK)"
            direction TB
            Interface[Engine Facade]
            
            subgraph "Logic Core"
                SeatMgr[Seat Manager]
                Algo[Bitmap Algorithms]
            end
            
            subgraph "Drivers"
                RedisDriver[Redis Bitmap Impl]
                MemDriver[Memory Impl (Test)]
            end
        end
        
        API --> |Import & Call| Interface
        Interface --> SeatMgr
        SeatMgr --> RedisDriver
    end

    subgraph "Infrastructure"
        Redis[(Redis Cluster)]
    end

    subgraph "Risk Zone (Sidecar)"
        PyRisk[Python Risk Engine]
    end

    RedisDriver --> |BitOps| Redis
    API -.-> |Async Check| PyRisk
```

#### 2. 组件解构 (Component Anatomy)

**A. Galaxy-Engine (SDK)**
*   **单一职责**：只负责“位置状态”的管理（占用、释放、查询）。不负责支付、不负责用户登录。
*   **通信协议**：Go Interface (In-process call)。
*   **状态管理**：**无状态 (Stateless)**。所有状态下沉到 Redis Bitmap。

**B. Galaxy-SaaS (Host App)**
*   **单一职责**：业务组装。负责 Auth、订单支付流程、数据持久化（MySQL）。
*   **通信协议**：HTTP/WebSocket 对外。

**C. Risk-Sidecar (Python)**
*   **单一职责**：复杂计算。计算 IP 熵值、行为轨迹异动。

#### 3. 难点预警 (Risk Analysis)
*   **依赖注入地狱**：SDK 不能写死 Redis 配置，必须设计优雅的 `Option` 模式让用户注入 `redis.Client`。
*   **Lua 脚本维护**：Lua 脚本嵌在 Go 字符串里很难调试。需要建立“脚本加载机制”，支持 SHA1 缓存执行。
*   **SaaS 数据一致性**：SDK 在 Redis 扣减成功了，但 SaaS 层 MySQL 写入失败怎么办？（需要 SaaS 层实现最终一致性补偿任务）。

---

### ⚔️ 第三步：实战演练 (Execution & Engineering)

我们将优先构建 **Galaxy-Engine (SDK)**，因为它是 SaaS 的基石。

#### 3.1 粮草先行：环境与依赖

**SDK 依赖配置 (`galaxy-engine/go.mod`)**:

```go
module github.com/yourname/galaxy-engine

go 1.21

require (
    github.com/redis/go-redis/v9 v9.4.0 // 核心依赖：Redis 客户端
    github.com/stretchr/testify v1.8.4  // 核心依赖：单元测试
)
```

#### 3.2 安营扎寨：工程目录结构

我们需要一个 Monorepo 结构，或者将 SDK 独立。这里为了演示方便，假设在同一仓库下。

```bash
# 执行此脚本构建骨架
mkdir -p galaxy-ticket/{engine,saas,risk,deploy}

# 1. SDK 核心库结构
cd galaxy-ticket/engine
mkdir -p internal/{lua,storage,algo} pkg/model
touch engine.go options.go go.mod

# 2. SaaS 宿主结构
cd ../saas
mkdir -p cmd/api internal/{handler,service,repo}
touch main.go go.mod

# 3. Python 风控
cd ../risk
mkdir -p src tests
```

#### 3.3 阵地攻坚：核心代码交付 (SDK Core)

**目标**：实现一个通用的、可注入 Redis 的原子选座引擎。

**File**: `engine/engine.go` (门面接口)

```go
package engine

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourname/galaxy-engine/internal/storage"
)

var (
	ErrSeatUnavailable = errors.New("seat is unavailable")
	ErrConfigMissing   = errors.New("redis client is required")
)

// Galaxy 定义了 SDK 的核心能力
type Galaxy interface {
	// LockSeat 尝试原子锁定一个座位
	// eventID: 场次唯一标识
	// seatIndex: 座位的线性索引 (row * width + col)
	// userID: 用户标识
	LockSeat(ctx context.Context, eventID string, seatIndex int64, userID string) (bool, error)
	
	// ReleaseSeat 释放座位（一般由 SaaS 层的支付超时或取消订单触发）
	ReleaseSeat(ctx context.Context, eventID string, seatIndex int64) error
}

type galaxyImpl struct {
	store storage.Provider
	opts  *Options
}

// NewGalaxy 初始化引擎
func NewGalaxy(opts ...Option) (Galaxy, error) {
	o := defaultOptions()
	for _, fn := range opts {
		fn(o)
	}

	if o.redisClient == nil {
		return nil, ErrConfigMissing
	}

	return &galaxyImpl{
		store: storage.NewRedisProvider(o.redisClient),
		opts:  o,
	}, nil
}

func (g *galaxyImpl) LockSeat(ctx context.Context, eventID string, seatIndex int64, userID string) (bool, error) {
	// 委托给底层存储实现原子操作
	return g.store.Lock(ctx, eventID, seatIndex, userID, g.opts.LockTTL)
}

func (g *galaxyImpl) ReleaseSeat(ctx context.Context, eventID string, seatIndex int64) error {
	return g.store.Unlock(ctx, eventID, seatIndex)
}
```

**File**: `engine/options.go` (配置模式)

```go
package engine

import (
	"time"
	"github.com/redis/go-redis/v9"
)

type Options struct {
	redisClient *redis.Client
	LockTTL     time.Duration
}

type Option func(*Options)

func defaultOptions() *Options {
	return &Options{
		LockTTL: 5 * time.Minute,
	}
}

// WithRedis 允许用户注入自己的 Redis 连接池
func WithRedis(rdb *redis.Client) Option {
	return func(o *Options) {
		o.redisClient = rdb
	}
}

// WithLockTTL 设置默认锁座时间
func WithLockTTL(d time.Duration) Option {
	return func(o *Options) {
		o.LockTTL = d
	}
}
```

**File**: `engine/internal/storage/redis.go` (最硬核的 Lua 实现)

```go
package storage

import (
	"context"
	_ "embed" // 用于嵌入 Lua 脚本
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/atomic_lock.lua
var lockScript string

type Provider interface {
	Lock(ctx context.Context, eventID string, offset int64, userID string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, eventID string, offset int64) error
}

type redisProvider struct {
	client *redis.Client
}

func NewRedisProvider(client *redis.Client) Provider {
	return &redisProvider{client: client}
}

func (r *redisProvider) Lock(ctx context.Context, eventID string, offset int64, userID string, ttl time.Duration) (bool, error) {
	// Key 命名策略: galaxy:{event_id}:bitmap
	bitmapKey := fmt.Sprintf("galaxy:%s:seats", eventID)
	// Key 命名策略: galaxy:{event_id}:lock:{offset} -> userID
	lockDetailKey := fmt.Sprintf("galaxy:%s:lock:%d", eventID, offset)

	// 执行 Lua 脚本，保证 "检查位图 -> 修改位图 -> 设置详情Key" 是原子的
	res, err := r.client.Eval(ctx, lockScript, 
		[]string{bitmapKey, lockDetailKey}, // KEYS
		offset, userID, int(ttl.Seconds()), // ARGV
	).Result()

	if err != nil {
		return false, err
	}

	// Lua 返回 1 表示成功，0 表示已被抢
	return res.(int64) == 1, nil
}

func (r *redisProvider) Unlock(ctx context.Context, eventID string, offset int64) error {
	bitmapKey := fmt.Sprintf("galaxy:%s:seats", eventID)
	lockDetailKey := fmt.Sprintf("galaxy:%s:lock:%d", eventID, offset)

	// 简单的释放逻辑：将位图置0，删除锁详情
	// 实际生产中可能需要验证 UserID，这里简化
	pipe := r.client.Pipeline()
	pipe.SetBit(ctx, bitmapKey, offset, 0)
	pipe.Del(ctx, lockDetailKey)
	_, err := pipe.Exec(ctx)
	return err
}
```

**File**: `engine/internal/storage/lua/atomic_lock.lua` (Lua 脚本)

```lua
-- KEYS[1]: bitmap key
-- KEYS[2]: lock detail key
-- ARGV[1]: offset
-- ARGV[2]: user_id
-- ARGV[3]: ttl_seconds

local offset = tonumber(ARGV[1])
local ttl = tonumber(ARGV[3])

-- 1. 检查 Bitmap 该位是否为 1
local is_taken = redis.call('GETBIT', KEYS[1], offset)

if is_taken == 1 then
    return 0 -- 失败
end

-- 2. 只有为 0 时才执行锁定
redis.call('SETBIT', KEYS[1], offset, 1)
redis.call('SET', KEYS[2], ARGV[2], 'EX', ttl)

return 1 -- 成功
```

#### 3.4 实弹演习：测试与验证 (User Integration Test)

我们不写常规的 Unit Test，我们写一个 **Integration Test**，模拟用户如何使用这个 SDK。

**File**: `engine/tests/integration_test.go`

```go
package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourname/galaxy-engine" // 引用你的 SDK
)

// 假设本地有 Docker Redis: docker run -p 6379:6379 redis
func TestHighConcurrencyBooking(t *testing.T) {
	// 1. 用户准备环境
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	
	// 2. 初始化 SDK
	svc, err := engine.NewGalaxy(
		engine.WithRedis(rdb),
		engine.WithLockTTL(10*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to init engine: %v", err)
	}

	eventID := "jay_chou_world_tour"
	seatIdx := int64(999)
	
	// 清理脏数据
	rdb.Del(context.Background(), fmt.Sprintf("galaxy:%s:seats", eventID))

	// 3. 模拟 50 个协程同时抢同一个座位
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			user := fmt.Sprintf("fan_%d", uid)
			
			// 核心调用
			ok, _ := svc.LockSeat(context.Background(), eventID, seatIdx, user)
			
			if ok {
				mu.Lock()
				successCount++
				mu.Unlock()
				t.Logf("User %s GOT the seat!", user)
			}
		}(i)
	}

	wg.Wait()

	// 4. 断言：绝对只能有 1 个人抢到
	if successCount != 1 {
		t.Fatalf("Consistency Error! Expected 1 winner, but got %d", successCount)
	}
	t.Log("Integration Test Passed: System is atomic.")
}
```

---

### 🔮 第四步：大匠运斤 (Evolution & Ecosystem)

现在你拥有了一个坚如磐石的内核（SDK）。为了让它成为 1k Stars 项目，我们需要在后续完善以下 SaaS 层的特性：

1.  **可视化控制台 (SaaS Side)**：
    *   在 SaaS 项目中，编写一个 API 接口 `GET /seats/bitmap`，直接返回 Redis Bitmap 的 Base64 字符串。前端解析这个字符串，直接渲染 Canvas，性能极高。
2.  **智能连坐算法 (SDK Upgrade)**：
    *   在 `engine/algo` 中实现位操作算法。比如查找 "连续3个0"。
    *   Go 伪代码：`current_bitmap & (current_bitmap << 1) & (current_bitmap << 2)`，结果为 0 的位置即为连续空位。
3.  **风控接入**：
    *   在 SaaS 层调用 SDK 前，先发一个 gRPC 请求给 Python Sidecar，如果 Python 返回 `RiskScore > 80`，则直接拦截，不调用 SDK。

**架构师结语**：
现在，你的项目结构已经非常清晰。
*   `galaxy-engine`: 纯粹、高性能、无依赖的 Go 库（这是你的技术名片）。
*   `galaxy-saas`: 展示如何使用该库构建业务系统的 Demo。

**下一步指令**：
如果你准备好开始编写 **SaaS 层的 API 接口（Gin 框架）** 以及 **前端 Canvas 交互逻辑**，请回复 **“启动 SaaS 建设”**。
如果你想深入 **Python 风控大脑的实现细节**，请回复 **“部署风控哨兵”**。