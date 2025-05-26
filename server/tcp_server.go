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

func (m *manager) initTcpListener() {
	conf := config.GetConf()

	listener, err := net.Listen("tcp", conf.TcpListenerAddress)
	if err != nil {
		log.Panic("[tcp_server] 初始化tcp监听服务失败", zap.Error(err))
	}
	m.tcpListener = listener
}

func (m *manager) tcpAccept(ctx context.Context) {
	addr := m.tcpListener.Addr()
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	log.Info("[tcp_server]  开启tcp监听服务", zap.Any("addr", addr.String()))
	for m.isRun.Load() {
		// 接受客户端的连接
		conn, err := m.tcpListener.Accept()
		if err != nil {
			// 若接受连接出错，记录错误信息并继续等待下一个连接
			log.Error("[tcp_server]  tcp监听服务Accept", zap.Error(err), zap.Any("addr", addr.String()))
			continue
		}
		wg.Add(1)
		m.tcpGoCount.Add(1)
		go func() {
			defer wg.Done()
			defer m.tcpGoCount.Add(-1)
			m.handlerTcpConn(ctx, conn)
			log.Info("[tcp_server] 退出")
		}()
	}
}

func (m *manager) responseClient(req *http.Request, conn net.Conn, targetConn net.Conn) error {
	if req.Method == http.MethodConnect {
		if _, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
			return fmt.Errorf("响应客户端Connection Established失败%+v", err)
		}
		return nil
	} else {
		req.Header.Del("Proxy-Authorization")
		var buf bytes.Buffer
		if err := req.Write(&buf); err != nil {
			return fmt.Errorf("清除Proxy-Authorization失败%+v", err)
		}

		if _, err := targetConn.Write(buf.Bytes()); err != nil {
			return fmt.Errorf("转发http请求数据失败%+v", err)
		}
	}
	return nil
}

func (m *manager) shakeHands(ctx context.Context, conn net.Conn) (targetConn net.Conn, err error) {
	req, err := m.getRequest(ctx, conn)
	if err != nil {
		err = fmt.Errorf("解析http请求头%+v", err)
		return
	}

	err = m.auth(ctx, req)
	if err != nil {
		err = fmt.Errorf("鉴权失败%+v", err)
		return
	}

	address := req.Host
	_, port, _ := net.SplitHostPort(req.Host)
	if req.Method == "CONNECT" {
		if port == "" {
			address = fmt.Sprint(req.Host, ":", 443)
		}
	} else {
		if port == "" {
			address = fmt.Sprint(req.Host, ":", 80)
		}
	}

	targetConn, err = m.connection2target(ctx, address)
	if err != nil {
		err = fmt.Errorf("连接目标地址%s失败%+v", address, err)
		return
	}

	var eRR error = nil
	c := utils.DoOrCancel(ctx.Done(), func() {
		targetConn.Close()
		conn.Close()
		eRR = fmt.Errorf("握手超时")
	})
	defer func() {
		c()
		if eRR != nil {
			err = eRR
		}
	}()

	err = m.responseClient(req, conn, targetConn)
	if err != nil {
		return
	}

	return
}

// / shakeHands
func (m *manager) getRequest(ctx context.Context, conn net.Conn) (*http.Request, error) {
	buffer := m.bufPool.Get().([]byte)
	defer m.bytePool.Put(buffer)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	bufferReader := bytes.NewBuffer(buffer[:n])
	reader := bufio.NewReader(bufferReader)
	req, err := http.ReadRequest(reader)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (m *manager) auth(ctx context.Context, req *http.Request) error {
	return nil
}

func (m *manager) connection2target(ctx context.Context, address string) (net.Conn, error) {
	d := net.Dialer{}
	return d.DialContext(ctx, "tcp", address)
}

func (m *manager) handlerTcpConn(ctx context.Context, conn net.Conn) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	targetConn, err := m.shakeHands(ctxTimeout, conn)
	if err != nil {
		conn.Close()
		log.Error("[tcp_server]	shakeHands失败", zap.Error(err))
		return
	}
}
