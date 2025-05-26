package connContainer

type Conn interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	CheckoutIsConn() <-chan struct{}
	GetBufAndErr() ([]byte, error)
	Close() error
}
