package tunnel

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"go.uber.org/zap"
)

type connState struct {
	connId         int64
	nextWriteSeqno int64
	stream         stream      // stream it's associated with
	logger         *zap.Logger // can be nil

	closed   atomic.Bool
	closedCh chan struct{}

	sendMu sync.Mutex
	seqno  int64

	readsMu  sync.Mutex
	reads    chan []byte
	readsBuf []byte

	readDeadlineMu    sync.Mutex
	readDeadlineTimer *time.Timer
	readDeadlineChan  chan struct{}
}

var _ net.Conn = (*connState)(nil)

func newConnState(streamm stream, connId int64, secret []byte, logger *zap.Logger) *connState {
	if logger != nil {
		logger = logger.Named("connState").With(
			zap.Int64("connId", connId))
	}

	var s stream = streamm
	// if len(secret) > 0 {
	// 	s = newCryptedStream(streamm, secret)
	// }

	return &connState{
		connId: connId,
		seqno:  0,
		stream: s,
		logger: logger,

		closedCh: make(chan struct{}),

		reads:    make(chan []byte, 10), // Buffered channel to hold a few writes worth of messages before blocking
		readsBuf: nil,

		readDeadlineChan: make(chan struct{}),
	}
}

// sendOpenPacket is only sent by the end starting the connection, not by the end that's opening it in response to a receive.
func (c *connState) sendOpenPacket() error {
	if c.closed.Load() {
		return net.ErrClosed
	}

	if c.logger != nil {
		c.logger.Info("sending open packet")
	}

	return c.stream.Send(&v1sync.TunnelMessage{
		ConnId: c.connId,
		Seqno:  0, // Open packet has Seqno 0
	})
}

func (c *connState) Write(data []byte) (int, error) {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if c.closed.Load() {
		return 0, net.ErrClosed
	}

	c.seqno++
	if c.logger != nil {
		c.logger.Debug("writing data", zap.Int("dataLength", len(data)), zap.Int64("seqno", c.seqno))
	}
	err := c.stream.Send(&v1sync.TunnelMessage{
		ConnId: c.connId,
		Data:   bytes.Clone(data),
		Seqno:  c.nextWriteSeqno,
	})
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to send data", zap.Int64("connId", c.connId), zap.Error(err))
		}
		return 0, err
	}
	return len(data), nil
}

func (c *connState) Read(b []byte) (int, error) {
	c.readsMu.Lock()
	defer c.readsMu.Unlock()

	if len(c.readsBuf) > 0 {
		n := copy(b, c.readsBuf)
		c.readsBuf = c.readsBuf[n:]
		return n, nil
	}

	c.readDeadlineMu.Lock()
	readDeadlineChan := c.readDeadlineChan
	c.readDeadlineMu.Unlock()

	select {
	case data := <-c.reads:
		if len(data) == 0 {
			return 0, net.ErrClosed
		}
		if c.logger != nil {
			c.logger.Debug("conn state c.reads received packet", zap.Int("dataLength", len(data)))
		}
		n := copy(b, data)
		if n < len(data) {
			c.readsBuf = data[n:]
		}
		return n, nil
	case <-readDeadlineChan:
		if c.logger != nil {
			c.logger.Info("read deadline reached")
		}
		return 0, os.ErrDeadlineExceeded
	case <-c.closedCh:
		if c.logger != nil {
			c.logger.Info("connection closed while waiting for read")
		}
		return 0, net.ErrClosed
	}
}

func (c *connState) Close() error {
	if !c.closed.Swap(true) {
		if c.logger != nil {
			c.logger.Info("closing connection")
		}
		close(c.closedCh)
		if err := c.stream.Send(&v1sync.TunnelMessage{
			ConnId: c.connId,
			Close:  true,
		}); err != nil {
			if c.logger != nil {
				c.logger.Error("failed to send close message", zap.Error(err))
			}
			return fmt.Errorf("send close message: %w", err)
		}
		if c.logger != nil {
			c.logger.Info("connection closed successfully")
		}
	} else if c.logger != nil {
		c.logger.Warn("close called on already closed connection")
	}
	return nil
}

func (c *connState) LocalAddr() net.Addr {
	// Return the local address of the tunnel connection
	return &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}
}

func (c *connState) RemoteAddr() net.Addr {
	// Return the remote address of the tunnel connection
	return &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}
}

func (c *connState) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

func (c *connState) SetReadDeadline(t time.Time) error {
	c.readDeadlineMu.Lock()
	defer c.readDeadlineMu.Unlock()
	if c.readDeadlineTimer != nil {
		c.readDeadlineTimer.Stop()
	}
	if t.IsZero() {
		c.readDeadlineTimer = nil
		return nil
	}
	c.readDeadlineTimer = time.AfterFunc(time.Until(t), func() {
		close(c.readDeadlineChan)
	})
	c.readDeadlineChan = make(chan struct{}) // Reset the channel to ensure it is ready for the next deadline
	return nil
}

func (c *connState) SetWriteDeadline(t time.Time) error {
	// TODO: simply does not work right now
	return nil
}
