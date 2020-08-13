package cs

import (
	"errors"
	"net"
	"sync"

	"github.com/thinkgos/jocasta/connection/cencrypt"
	"github.com/thinkgos/jocasta/connection/csnappy"
	"github.com/thinkgos/jocasta/lib/encrypt"
	"github.com/thinkgos/jocasta/lib/gpool"
)

// StcpServer stcp server
type StcpServer struct {
	Addr     string
	Method   string
	Password string
	Compress bool
	Status   chan error
	GoPool   gpool.Pool
	Handler  Handler

	mu sync.Mutex
	ln net.Listener
}

// ListenAndServe listen and serve
func (sf *StcpServer) ListenAndServe() error {
	if sf.Method == "" || sf.Password == "" || !encrypt.HasCipherMethod(sf.Method) {
		err := errors.New("invalid method or password")
		setStatus(sf.Status, err)
		return err
	}
	_, err := encrypt.NewCipher(sf.Method, sf.Password)
	if err != nil {
		setStatus(sf.Status, err)
		return err
	}

	ln, err := net.Listen("tcp", sf.Addr)
	if err != nil {
		setStatus(sf.Status, err)
		return err
	}
	defer ln.Close()

	sf.mu.Lock()
	sf.ln = ln
	sf.mu.Unlock()

	setStatus(sf.Status, nil)
	for {
		conn, err := sf.ln.Accept()
		if err != nil {
			return err
		}
		gpool.Go(sf.GoPool, func() {
			if sf.Compress {
				conn = csnappy.New(conn)
			}
			// 这里应永远不出错,加密
			cip, _ := encrypt.NewCipher(sf.Method, sf.Password)
			sf.Handler.ServerConn(cencrypt.New(conn, cip))
		})
	}
}

// LocalAddr local listen address
func (sf *StcpServer) LocalAddr() (addr string) {
	sf.mu.Lock()
	if sf.ln != nil {
		addr = sf.ln.Addr().String()
	}
	sf.mu.Unlock()
	return
}

// Close close the server
func (sf *StcpServer) Close() (err error) {
	if sf.ln != nil {
		err = sf.ln.Close()
	}
	return
}