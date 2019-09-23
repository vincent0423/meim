package meim

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/ipiao/meim/log"
	"go.uber.org/atomic"
)

const (
	MessageQueueLimit = 1000
)

type Client struct {
	conn           Conn
	closed         atomic.Bool        // 是否关闭
	mch            chan *Message      // 一般消息下发通道, message channel
	lmsch          chan int           // 长消息下发(信号),long message signal channel
	lmessages      *list.List         // 长消息存储队列(非阻塞消息下发)
	extch          chan func(*Client) // 外部时间队列, external event channel
	mu             sync.Mutex         // 锁
	enqueueTimeout time.Duration      // 消息/事件入队超时时间

	UID      int64       // 用户id
	UserData interface{} // 用户其他私有数据
	DC       DataCreator // 协议数据构建器
}

func NewClient(conn Conn) *Client {
	client := new(Client)
	client.conn = conn
	client.mch = make(chan *Message, 16)
	client.lmsch = make(chan int, 1)
	client.lmessages = list.New()
	client.extch = make(chan func(*Client), 8)
	client.enqueueTimeout = time.Second * 10
	return nil
}

func (client *Client) Log() string {
	return fmt.Sprintf("uid %d, addr %s", client.UID, client.conn.RemoteAddr())
}

func (client *Client) String() string {
	return fmt.Sprintf(" uid: %d, addr: %s, data: %+v", client.UID, client.conn.RemoteAddr(), client.UserData)
}

// 发送一般消息
func (client *Client) EnqueueMessage(msg *Message) bool {
	if client.closed.Load() { // 已关闭
		log.Infof("can't send message to closed connection %s", client.Log())
		return false
	}

	select {
	case client.mch <- msg:
		return true
	case <-time.After(client.enqueueTimeout):
		log.Infof("send message to mch timed out %s", client.Log())
		return false
	}
}

// 发送非阻塞消息
func (client *Client) EnqueueNonBlockMessage(msg *Message) bool {
	if client.closed.Load() { // 已关闭
		log.Infof("can't send message to closed connection %s", client.Log())
		return false
	}

	dropped := false
	client.mu.Lock()
	if client.lmessages.Len() >= MessageQueueLimit {
		//队列阻塞，丢弃之前的消息
		client.lmessages.Remove(client.lmessages.Front())
		dropped = true
	}

	client.lmessages.PushBack(msg)
	client.mu.Unlock()

	if dropped {
		log.Info("connection %s message queue full, drop a message", client.Log())
	}

	//nonblock
	select {
	case client.lmsch <- 1:
	default:
	}
	return true
}

// 发送一般消息
func (client *Client) FlushMessage() {
	// for msg := range c.mch {
	// 	WriteMessage(c.conn, msg)
	// }
	client.SendLMessages()
}

//发送等待队列中的消息
func (client *Client) SendLMessages() {
	var messages *list.List
	client.mu.Lock()
	if client.lmessages.Len() == 0 {
		client.mu.Unlock()
		return
	}
	messages = client.lmessages
	client.lmessages = list.New()
	client.mu.Unlock()

	e := messages.Front()
	for e != nil {
		msg := e.Value.(*Message)
		WriteMessage(client.conn, msg)
		e = e.Next()
	}
}

func (client *Client) EnqueueEvent(fn func(*Client)) bool {
	if client.closed.Load() { // 已关闭
		log.Infof("can't add event to closed connection %s", client.Log())
		return false
	}

	select {
	case client.extch <- fn:
		return true
	case <-time.After(client.enqueueTimeout):
		log.Infof("add event to extch timed out %s", client.Log())
		return false
	}
}

func (client *Client) read() {
	for {
		msg, err := ReadLimitMessage(client.conn, client.DC, 128*1024)
		if err != nil {
			log.Info("client read error:", err)
			client.Close()
			break
		}
		ext.HandleMessage(client, msg)
	}
}

func (client *Client) write() {
	//发送在线消息
	for {
		select {
		case msg := <-client.mch:
			if msg == nil {
				log.Infof("client:%d socket closed", client.UID)
				client.FlushMessage()
				client.conn.Close()
				break
			}
			WriteMessage(client.conn, msg)

		case <-client.lmsch:
			client.SendLMessages()
			break

		}
	}
}

func (client *Client) Close() {
	if client.closed.CAS(false, true) {
		log.Infof("try close client %s", client.Log())
		client.mch <- nil
	}
}

func (client *Client) Run() {
	client.write()
	go client.read()
}