package tcpb

import (
	"errors"
	"net"
	"time"

	"github.com/ipiao/meim"

	"github.com/ipiao/meim/log"
)

type TCPBrokerClient struct {
	addr     string
	dc       meim.DataCreator
	conn     net.Conn
	subCmd   int
	unsubCmd int
}

func NewTCPRouterClient(addr string, dc meim.DataCreator, subCmd, unsubCmd int) *TCPBrokerClient {
	tr := &TCPBrokerClient{
		addr: addr,
		dc:   dc,
	}
	return tr
}

func (tr *TCPBrokerClient) Connect() {
	nsleep := 100
	for {
		conn, err := net.Dial("tcp", tr.addr)
		if err != nil {
			log.Infof("tcpRouter server error: %v", err)
			nsleep *= 2
			if nsleep > 60*1000 {
				nsleep = 60 * 1000
			}
			log.Infof("tcpRouter connect sleep: %d ms", nsleep)
			time.Sleep(time.Duration(nsleep) * time.Millisecond)
			continue
		}
		tconn := conn.(*net.TCPConn)
		tconn.SetKeepAlive(true)
		tconn.SetKeepAlivePeriod(time.Duration(10 * 60 * time.Second))
		log.Infof("tcpRouter connected")
		tr.conn = tconn
		return
	}
}

func (tr *TCPBrokerClient) SyncMessage(msg *meim.InternalMessage) (*meim.InternalMessage, error) {
	log.Warn("unsupported operation: SyncMessage")
	return nil, errors.New("SyncMessage not supported")
}

func (tr *TCPBrokerClient) SendMessage(msg *meim.InternalMessage) error {
	data, err := meim.EncodeInternalMessage(msg)
	if err == nil {
		_, err = tr.conn.Write(data)
	}
	return err
}

func (tr *TCPBrokerClient) ReceiveMessage() (*meim.InternalMessage, error) {
	return meim.ReadInternalMessage(tr.conn, tr.dc)
}

func (tr *TCPBrokerClient) Subscribe(uid int64) {
	msg := new(meim.InternalMessage)
	msg.Header = tr.dc.CreateHeader()
	msg.Header.SetCmd(tr.subCmd)
	msg.Sender = uid
	tr.SendMessage(msg)
}

func (tr *TCPBrokerClient) UnSubscribe(uid int64) {
	msg := new(meim.InternalMessage)
	msg.Header = tr.dc.CreateHeader()
	msg.Header.SetCmd(tr.unsubCmd)
	msg.Sender = uid
	tr.SendMessage(msg)
}

func (tr *TCPBrokerClient) Close() {
	tr.conn.Close()
}
