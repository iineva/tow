package towserver

import (
	"net/http"
	"strconv"
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
	sessMap    map[uint16]*Session
}

func NewServer() (*Server, error) {
	s := &Server{
		Logger:     chshare.NewLogger("server"),
		httpServer: chshare.NewHTTPServer(),
		sessMap:    make(map[uint16]*Session),
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
	sessionID, err := strconv.Atoi(r.Header.Get("Tow-Session-Id"))
	if err != nil {
		sessionID = 0
	}
	//websockets upgrade AND has chisel prefix
	if upgrade == "websocket" && protocol == towshare.ProtocolVersion {
		s.handleWS(w, r, uint16(sessionID))
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

func (s *Server) handleWS(w http.ResponseWriter, req *http.Request, sessionID uint16) {

	id := sessionID
	if sessionID == 0 {
		id = uint16(atomic.AddUint32(&s.sessCount, 1))
	}

	s.Infof("Websocket client did connected: %d", id)

	wsConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		s.Debugf("Failed to upgrade (%s)", err)
		return
	}

	if session, ok := s.sessMap[id]; ok {
		// reset wsConn
		session.SetWebSocketConn(towshare.NewWebSocketConn(wsConn))
	} else {
		// open new session
		session := NewSession(id, s.Fork("session#%d", id), towshare.NewWebSocketConn(wsConn))
		session.Write(towshare.MakeGetIdPackage(id))
		session.Start()
		s.sessMap[id] = session
	}

}
