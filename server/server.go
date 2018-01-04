package towserver

import (
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/iineva/tow/share"
	"github.com/jpillora/chisel/share"
)

type Server struct {
	*chshare.Logger
	httpServer *chshare.HTTPServer
	sessCount  uint32
}

func NewServer() (*Server, error) {
	s := &Server{
		Logger:     chshare.NewLogger("server"),
		httpServer: chshare.NewHTTPServer(),
	}
	s.Info = true
	s.Debug = true
	return s, nil
}

func (s *Server) Run(bindAddr string) error {
	s.Infof("Server start: http://%s", bindAddr)

	if err := s.Start(bindAddr); err != nil {
		return err
	}
	return s.Wait()
}

func (s *Server) Start(bindAddr string) error {

	h := http.Handler(http.HandlerFunc(s.handleHTTP))

	return s.httpServer.GoListenAndServe(bindAddr, h)
}

func (s *Server) Wait() error {
	return s.httpServer.Wait()
}

func (s *Server) Close() error {
	//this should cause an error in the open websockets
	return s.httpServer.Close()
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))
	protocol := r.Header.Get("Sec-WebSocket-Protocol")
	//websockets upgrade AND has chisel prefix
	if upgrade == "websocket" && protocol == towshare.ProtocolVersion {
		s.handleWS(w, r)
		return
	}

	//missing :O
	w.WriteHeader(404)
	w.Write([]byte("Not found"))
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (s *Server) handleWS(w http.ResponseWriter, req *http.Request) {

	id := atomic.AddUint32(&s.sessCount, 1)

	s.Infof("Websocket client did connected: %d", id)

	wsConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		s.Debugf("Failed to upgrade (%s)", err)
		return
	}

	session := NewSession(id, s.Fork("session#%d", id), towshare.NewWebSocketConn(wsConn))

	// open connect
	session.Start()
}
