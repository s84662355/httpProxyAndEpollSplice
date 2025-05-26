package server

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap" // 高性能日志库

	"Hepoll/config"
	"Hepoll/connContainer"
	"Hepoll/log"
	"Hepoll/server/relayConn"
	"Hepoll/utils"
)

// 添加文件描述符（线程安全）
func (m *EpollManager) AddFD(fd int, events uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	unix.EpollCtl(m.epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: events,
		Fd:     int32(fd),
	})
}

// 删除文件描述符（线程安全）
func (m *EpollManager) DelFD(fd int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	unix.EpollCtl(m.epfd, unix.EPOLL_CTL_DEL, fd, nil)
}

// 多个协程可以同时调用 EpollWait
func (m *EpollManager) StartWorker() {
	go func() {
		events := make([]unix.EpollEvent, 100)
		for {
			n, err := unix.EpollWait(m.epfd, events, -1)
			// 处理事件...
		}
	}()
}
