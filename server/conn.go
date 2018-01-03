package towserver

import (
	"fmt"
	"net"

	"github.com/iineva/tow/share"
	"github.com/jpillora/chisel/share"
)

type Conn struct {
	Logger     *chshare.Logger
	Id         uint16
	socketConn net.Conn // socket connect
	Session    *Session
	Remote     string
	RemoteConn net.Conn
	running    bool
}

func NewConn(id uint16, remote string, session *Session) *Conn {
	log := chshare.NewLogger(fmt.Sprintf("conn#%d", int(id)))
	log.Info = true
	log.Debug = true
	return &Conn{
		Id:      id,
		Remote:  remote,
		Session: session,
		running: true,
		Logger:  log,
	}
}

func (c *Conn) Open() error {
	c.Logger.Infof("TCP Connect start!")
	dst, err := net.Dial("tcp", c.Remote)
	if err != nil {
		c.Logger.Debugf("Remote failed (%s)", err)
		c.SendError("Remote failed ID: %s", c.Id)
		return err
	}
	c.RemoteConn = dst
	go (func() {
		for {
			if !c.running {
				break
			}

			buf := make([]byte, 1024)
			l, err := dst.Read(buf)
			if err != nil {
				c.SendError("Read remote stream error: %s", err)
				c.Close()
			}

			c.Session.Write(towshare.MakeDataPackage(c.Id, buf[0:l]))
		}
	})()
	c.Logger.Infof("TCP Connected!")
	return nil
}

func (c *Conn) Close() error {
	c.running = false
	c.SendClose()
	return c.RemoteConn.Close()
}

func (c *Conn) Write(b []byte) (int, error) {
	return c.RemoteConn.Write(b)
}

func (c *Conn) SendError(str string, s ...interface{}) error {
	_, err := c.Session.Write(towshare.MakeErrorPackage(c.Id, fmt.Sprintf(str, s...)))
	return err
}

func (c *Conn) SendClose() error {
	_, err := c.Session.Write(towshare.MakeClosePackage(c.Id))
	return err
}
