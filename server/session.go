package towserver

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/iineva/tow/share"
	"github.com/jpillora/chisel/share"
)

type Session struct {
	io.ReadWriteCloser
	Id                  uint16
	Logger              *chshare.Logger
	webSocketConn       net.Conn // websocket connect
	webSocketReadMutex  sync.Mutex
	webSocketWriteMutex sync.Mutex
	running             bool
	closed              bool
	Conns               map[uint16]*Conn
	keepAlive           time.Duration
}

type SessionError struct {
	error
	s string
}

func (s *SessionError) Error() string {
	return s.s
}
func NewSessionError(s string) *SessionError {
	return &SessionError{s: s}
}

func NewSession(id uint16, log *chshare.Logger, conn net.Conn) *Session {
	s := &Session{
		Id:            id,
		Logger:        log,
		webSocketConn: conn,
		running:       false,
		closed:        false,
		Conns:         map[uint16]*Conn{},
		keepAlive:     1 * time.Second,
	}
	s.Logger.Info = true
	s.Logger.Debug = true

	// keep alive
	go func() {
		for range time.Tick(s.keepAlive) {
			if s.running {
				s.Logger.Debugf("Send keep alive package...")
				s.Write(towshare.MakeKeepAlivePackage())
			} else if s.closed {
				break
			}
		}
	}()

	return s
}

func (s *Session) Start() {

	s.running = true

	// read data
	go (func() {
		for {

			if !s.running {
				break
			}

			buf := make([]byte, 1024*6+10) // max buffer 6k
			l, err := s.webSocketConn.Read(buf)

			if err != nil {
				s.Logger.Debugf("Read stream error: %s", err)

				s.running = false
				s.webSocketConn.Close()
				break
			}

			buf = buf[:l]

			if len(buf) == 0 {
				continue
			}

			p := towshare.NewPackage(buf)

			s.Logger.Debugf("Session get package: %d", p.Type)

			// handle package
			s.handlePackage(p)
		}
	})()
}

func (s *Session) handlePackage(p *towshare.Package) error {
	switch p.Type {
	case towshare.PackageTypeOpen:
		// new connect
		conn := NewConn(p.Id, p.Remote, s)
		if s.Conns[p.Id] != nil {
			s.Conns[p.Id].Close()
		}
		s.Conns[p.Id] = conn
		conn.Open()
	case towshare.PackageTypeData:
		for _, v := range s.Conns {
			if v.Id == p.Id {
				v.Write(p.Payload)
				break
			}
		}
	case towshare.PackageTypeClose:
	case towshare.PackageTypeError:
		for _, v := range s.Conns {
			if v.Id == p.Id {
				v.Close()
				delete(s.Conns, p.Id)
				break
			}
		}
	case towshare.PackageTypeAlive:
		// do nothing
		break
	}
	return nil
}

// reconnect session
func (s *Session) SetWebSocketConn(conn net.Conn) error {

	s.webSocketReadMutex.Lock()
	s.webSocketWriteMutex.Lock()
	defer s.webSocketReadMutex.Unlock()
	defer s.webSocketWriteMutex.Unlock()

	err := s.webSocketConn.Close()
	s.webSocketConn = conn
	s.Start()

	return err
}

func (s *Session) Close() error {
	s.running = false
	s.closed = true
	err := s.webSocketConn.Close()
	if err != nil {
		return err
	}
	for _, v := range s.Conns {
		err := v.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) Read(b []byte) (n int, err error) {
	s.webSocketReadMutex.Lock()
	defer s.webSocketReadMutex.Unlock()
	return s.webSocketConn.Read(b)
}

func (s *Session) Write(b []byte) (n int, err error) {
	s.webSocketWriteMutex.Lock()
	defer s.webSocketWriteMutex.Unlock()
	return s.webSocketConn.Write(b)
}
