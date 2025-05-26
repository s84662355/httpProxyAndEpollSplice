package connContainer

import (
	"io"
)

func CopyBuffer(dst Conn, src Conn, buf []byte) (written int64, err error) {
	{
		<-src.CheckoutIsConn()
		buft, err := src.GetBufAndErr()
		if err != nil {
			return 0, err
		}

		if len(buft) > 0 {
			n, err := dst.Write(buft)
			if err != nil {
				return 0, err
			}
			written += int64(n)
		}

	}

	return io.CopyBuffer(dst, src, buf)
}
