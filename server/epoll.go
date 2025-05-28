package server

import (
	"context"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap" // 高性能日志库

	"Hepoll/config"
	"Hepoll/connContainer"
	"Hepoll/log"
	"Hepoll/server/relayConn"
	"Hepoll/utils"
)

type FdConn struct {
	conn  net.Conn
	dstFd int
}

// 添加文件描述符（线程安全）
func (m *EpollManager) AddFD(src, dst net.Conn) error {
	m.epfdMu.Lock()
	defer m.epfdMu.Unlock()

	if srcFd, err := netFd(src); err == nil {
		if dstFd, err := netFd(dst); err == nil {

			err = unix.EpollCtl(m.epfd, unix.EPOLL_CTL_ADD, src, &unix.EpollEvent{
				Events: uint32(uintptr(unix.EPOLLIN)),
				Fd:     int32(srcFd),
				Pad:    int32(dstFd),
			})
			if err != nil {
				return err
			}

			m.epollFdMap.Upsert(srcFd, nil, func(exists bool, valueInMap *FdConn, newValue *FdConn) *FdConn {
				return &FdConn{
					conn:  src,
					dstFd: dstFd,
				}
			})

		} else {
			return err
		}
	} else {
		return err
	}

	// 将客户端套接字添加到 epoll 实例中
	// err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(cfd.Fd()), &unix.EpollEvent{
	// 	//    Events: uint32(uintptr(unix.EPOLLIN | unix.EPOLLET)),
	// 	Events: uint32(uintptr(unix.EPOLLIN)),
	// 	Fd:     int32(cfd.Fd()),
	// 	Pad:    int32(targetFd),
	// })
}

// 删除文件描述符（线程安全）
func (m *EpollManager) DelFD(src net.Conn) {
	m.epfdMu.Lock()
	defer m.epfdMu.Unlock()
	if srcFd, err := netFd(src); err == nil {
		// Monkey should be removed
		result := m.RemoveCb(srcFd, func(key int, val *FdConn, exists bool) bool {
			if exists && val.src == conn {
				if err:=unix.EpollCtl(m.epfd, unix.EPOLL_CTL_DEL, srcFd, nil);err!=nil{
					return false
				}
				conn.Close()
				return true

			}

			return false
		})
	}
}

// 多个协程可以同时调用 EpollWait
func (m *EpollManager) StartWorker(ctx context.Context) {
	events := make([]unix.EpollEvent, 100)
	for m.isRun.Load() {
		select {
		case <-ctx.Done():
			return
		default:
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
	}
}

func netFd(conn net.Conn) (int, error) {
	syscallConn, ok := conn.(syscall.Conn)
	if !ok {
		return 0, ErrSyscallConn
	}
	return fd(syscallConn)
}

func fd(c syscall.Conn) (int, error) {
	var nfd int
	raw, err := c.SyscallConn()
	if err != nil {
		return 0, err
	}
	raw.Control(func(fd uintptr) {
		nfd = int(fd)
	})
	return nfd, nil
}
