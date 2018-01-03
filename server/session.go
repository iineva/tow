package towserver

import (
	"io"
	"net"
	"sync"

	"github.com/iineva/tow/share"
	"github.com/jpillora/chisel/share"
)

type Session struct {
	io.ReadWriteCloser
	Id                  uint32
	Logger              *chshare.Logger
	webSocketConn       net.Conn // websocket connect
	webSocketReadMutex  sync.Mutex
	webSocketWriteMutex sync.Mutex
	running             bool
	Conns               map[uint16]*Conn
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

func NewSession(id uint32, log *chshare.Logger, conn net.Conn) *Session {
	session := &Session{
		Id:            id,
		Logger:        log,
		webSocketConn: conn,
		running:       true,
		Conns:         map[uint16]*Conn{},
	}
	session.Logger.Info = true
	session.Logger.Debug = false
	return session
}

func (s *Session) Start() {

	// read data
	// TODO: thread save
	go (func() {
		for {

			if !s.running {
				break
			}

			buf := make([]byte, 1024*6+10) // max buffer 6k
			l, err := s.webSocketConn.Read(buf)

			if err != nil {
				s.Logger.Debugf("Read stream error: %s", err)
				s.Close()
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
	}
	return nil
}

func (s *Session) Close() error {
	s.running = false
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
