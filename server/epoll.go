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
func (m *EpollManager) AddFD(src, dst int) {
	m.epfdMu.Lock()
	defer m.epfdMu.Unlock()
	unix.EpollCtl(m.epfd, unix.EPOLL_CTL_ADD, src, &unix.EpollEvent{
		Events: uint32(uintptr(unix.EPOLLIN)),
		Fd:     int32(src),
		Pad:    int32(dst),
	})

	// 将客户端套接字添加到 epoll 实例中
	// err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(cfd.Fd()), &unix.EpollEvent{
	// 	//    Events: uint32(uintptr(unix.EPOLLIN | unix.EPOLLET)),
	// 	Events: uint32(uintptr(unix.EPOLLIN)),
	// 	Fd:     int32(cfd.Fd()),
	// 	Pad:    int32(targetFd),
	// })
}

// 删除文件描述符（线程安全）
func (m *EpollManager) DelFD(src int) {
	m.epfdMu.Lock()
	defer m.epfdMu.Unlock()
	unix.EpollCtl(m.epfd, unix.EPOLL_CTL_DEL, src, nil)
}

// 多个协程可以同时调用 EpollWait
func (m *EpollManager) StartWorker() {
	go func() {
		events := make([]unix.EpollEvent, 100)
		for m.isRun.Load() {
			n, err := unix.EpollWait(m.epfd, events, 100)
			if err != nil {
				log.Error("[epoll] EpollWait", zap.Error(err))
				continue
			}

			for i := 0; i < n; i++ {
				// 有数据可读，进行数据复制
				srcFd := int(event.Fd)
				dstFd := int(event.Pad) // 这里需要根据实际情况找到对应的目标套接字

			}

		}
	}()
}
