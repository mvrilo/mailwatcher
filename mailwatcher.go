package mailwatcher

import (
	"bytes"
	"net/mail"
	"time"

	"code.google.com/p/go-imap/go1/imap"
)

// Response with header, uid and body of the mail
type Message struct {
	Header mail.Header
	Body   []byte
	UID    uint32
}

type Mail struct {
	User    string
	Pass    string
	Address string
	Mailbox string
	num     uint32
	lastUID uint32
	client  *imap.Client
	cmd     *imap.Command
}

// Initialize new Mail
func New(user, pwd, addr string) (*Mail, error) {
	m := &Mail{
		User:    user,
		Pass:    pwd,
		Address: addr,
		Mailbox: "INBOX",
		num:     1,
	}
	return m.Start()
}

// Runs dial and login
func (m *Mail) Start() (*Mail, error) {
	_, err := m.dial()
	if err != nil {
		return nil, err
	}
	_, err = m.login()
	if err != nil {
		return nil, err
	}
	return m, err
}

func (m *Mail) dial() (c *imap.Client, err error) {
	c, err = imap.DialTLS(m.Address, nil)
	m.client = c
	return
}

func (g *Mail) login() (*imap.Command, error) {
	g.client.StartTLS(nil)
	return g.client.Login(g.User, g.Pass)
}

// Fetch the latest email using the field Mailbox and an argument for a filter (e.g. "UNSEEN") to be searched
func (m *Mail) Fetch(filter string) (err error) {
	if m.cmd, err = imap.Wait(m.client.Select(m.Mailbox, false)); err != nil {
		return
	}

	if filter != "" {
		if m.cmd, err = imap.Wait(m.client.UIDSearch(filter)); err != nil {
			return
		}
	}

	set, _ := imap.NewSeqSet("")
	set.AddNum(m.client.Mailbox.Messages - m.num)
	if m.cmd, err = imap.Wait(m.client.Fetch(set, "RFC822.HEADER", "UID", "BODY[1]")); err != nil {
		return
	}
	return
}

func (m *Mail) parseResponse(rsp *imap.Response) (uint32, []byte, *mail.Message, error) {
	info := rsp.MessageInfo()
	body := imap.AsBytes(info.Attrs["BODY[1]"])
	header := bytes.NewReader(imap.AsBytes(info.Attrs["RFC822.HEADER"]))
	hdr, err := mail.ReadMessage(header)
	if err != nil {
		return 0, nil, nil, err
	}
	return info.UID, body, hdr, nil
}

func (m *Mail) Messages() (msg []Message) {
	for _, rsp := range m.cmd.Data {
		uid, body, header, err := m.parseResponse(rsp)
		if err != nil {
			continue
		}
		msg = append(msg, Message{
			Header: header.Header,
			Body:   body,
			UID:    uid,
		})
	}
	return
}

// Blocks by looping with an specific interval and passes new incoming emails (inbox/unseen)
func (m *Mail) Watch(incoming chan Message, timer time.Duration) {
	ticker := time.NewTicker(time.Second * timer)
	for _ = range ticker.C {
		m.Fetch("UNSEEN")
		msgs := m.Messages()
		if len(msgs) == 0 {
			continue
		}

		msg := msgs[0]
		if m.lastUID == uint32(0) {
			m.lastUID = msg.UID
			continue
		}

		if m.lastUID != msg.UID {
			m.lastUID = msg.UID
			incoming <- msg
		}
	}
}

// Same as Watch func, but using a callback instead of passing to a channel
func (m *Mail) WatchFunc(timer time.Duration, fn func(Message)) {
	incoming := make(chan Message, 1)
	go m.Watch(incoming, timer)
	for {
		msg := <-incoming
		fn(msg)
	}
}
