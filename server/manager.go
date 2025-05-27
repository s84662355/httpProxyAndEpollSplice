package server

import (
	"context"
	"net"
	"relay/log"
	"relay/utils/taskConsumerManager"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"

	"github.com/orcaman/concurrent-map/v2"
	"go.uber.org/zap" // 高性能日志库
)

const AcceptAmount = 8

// 使用 sync.OnceValue 确保 manager 只被初始化一次（线程安全）
var newManager = sync.OnceValue(func() *manager {
	m := &manager{
		tcm:            taskConsumerManager.New(), // 任务消费者管理器
		relayContainer: cmap.New[*connContainer.Container](),
	}

	m.bufPool = sync.Pool{
		New: func() any {
			return make([]byte, 1024)
		},
	}

	return m
})

// manager 结构体管理整个代理服务的核心组件
type manager struct {
	tcm                  *taskConsumerManager.Manager // 任务调度管理器
	epollFdMap           cmap.ConcurrentMap[int, *FdConn]
	tcpListener          net.Listener
	websocketTcpListener net.Listener
	isRun                atomic.Bool
	tcpGoCount           atomic.Int64
	webSocketGoCount     atomic.Int64
	bufPool              sync.Pool
	epfd                 int
	epfdMu               RWMutex
}

// Start 启动代理服务的各个组件
func (m *manager) Start() error {
	m.isRun.Store(true)

	m.epollFdMap = cmap.NewWithCustomShardingFunction[int, *FdConn](func(k int) uint32 {
		return uint32(k)
	})

	epfd, err := unix.EpollCreate1(0)
	if err != nil {
		log.Error("[manager]创建epoll失败", zap.Error(err))
		return err
	}
	m.epfd = epfd

	m.initTcpListener()
	m.tcm.AddTask(AcceptAmount, func(ctx context.Context) {
		m.tcpAccept(ctx)
		tc, _ := context.WithTimeout(ctx, 1*time.Second)
		<-tc.Done()
	})

	// m.tcm.AddTask(1, func(ctx context.Context) {
	// 	// 每 1 秒执行一次任务，直到 context 被取消
	// 	ticker := time.NewTicker(30 * time.Second)
	// 	defer ticker.Stop()

	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		case <-ticker.C:

	// 			log.Debug("[server_manager] 任务协程数量",
	// 				zap.Any("容器", m.relayContainer.Count()),
	// 				zap.Any("tcp协程", m.tcpGoCount.Load()),
	// 				zap.Any("webSocket协程", m.webSocketGoCount.Load()),
	// 			)
	// 		}
	// 	}
	// })

	return nil
}

// Stop 停止所有服务组件
func (m *manager) Stop() {
	m.isRun.Store(false)
	m.tcpListener.Close()
	m.tcm.Stop() // 停止任务消费者管理器，会触发所有任务的优雅关闭
	m.websocketTcpListener.Close()
}

func (m *manager) addServerConnToRelayContainer(wsk string, conn connContainer.Conn) (res *connContainer.Container) {
	m.relayContainer.Upsert(wsk, nil, func(exist bool, valueInMap *connContainer.Container, newValue *connContainer.Container) *connContainer.Container {
		if exist {
			if valueInMap.SetServerConn(conn) {
				res = valueInMap
			} else {
				res = nil
			}
			return valueInMap
		}
		valueInMap = connContainer.NewServerConn(conn)
		res = valueInMap
		return valueInMap
	})
	return
}

func (m *manager) addClientConnToRelayContainer(wsk string, ws connContainer.Conn) (res *connContainer.Container) {
	m.relayContainer.Upsert(wsk, nil, func(exist bool, valueInMap *connContainer.Container, newValue *connContainer.Container) *connContainer.Container {
		if exist {
			if valueInMap.SetClientConn(ws) {
				res = valueInMap
			} else {
				res = nil
			}
			return valueInMap
		}
		valueInMap = connContainer.NewClientConn(ws)
		res = valueInMap
		return valueInMap
	})
	return
}

func (m *manager) removeConnToRelayContainer(wsk string, s *connContainer.Container) {
	m.relayContainer.RemoveCb(wsk, func(key string, val *connContainer.Container, exists bool) bool {
		if exists {
			return val == s
		}
		return false
	})
}
