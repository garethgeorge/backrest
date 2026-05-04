package syncapi

import (
	"crypto/rand"
	"sync"
	"testing"

	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/testutil"
)

// fakeStream is a pair of in-memory channels simulating a bidirectional transport.
type fakeStream struct {
	sendCh chan *v1sync.SyncStreamItem
	recvCh chan *v1sync.SyncStreamItem
}

func (f *fakeStream) Send(item *v1sync.SyncStreamItem) error {
	f.sendCh <- item
	return nil
}

func (f *fakeStream) Receive() (*v1sync.SyncStreamItem, error) {
	item := <-f.recvCh
	return item, nil
}

// newFakeStreamPair creates two connected fakeStreams (A's send is B's recv and vice versa).
func newFakeStreamPair() (*fakeStream, *fakeStream) {
	ab := make(chan *v1sync.SyncStreamItem, 16)
	ba := make(chan *v1sync.SyncStreamItem, 16)
	return &fakeStream{sendCh: ab, recvCh: ba}, &fakeStream{sendCh: ba, recvCh: ab}
}

// runHandshake performs the post-quantum KEM handshake between an initiator
// and responder over a fakeStream pair, returning the resulting sessions in
// (initiator, responder) order.
func runHandshake(t *testing.T) (initiatorSess, responderSess *cryptoutil.TransportSession) {
	t.Helper()
	recipient, pubBytes, err := cryptoutil.NewTransportRecipient()
	if err != nil {
		t.Fatalf("NewTransportRecipient: %v", err)
	}
	enc, respSess, err := cryptoutil.EncapsulateToTransport(pubBytes)
	if err != nil {
		t.Fatalf("EncapsulateToTransport: %v", err)
	}
	initSess, err := recipient.Decapsulate(enc)
	if err != nil {
		t.Fatalf("Decapsulate: %v", err)
	}
	return initSess, respSess
}

func TestEncryptedStream_RoundTrip(t *testing.T) {
	initSess, respSess := runHandshake(t)

	transportA, transportB := newFakeStreamPair()
	encA := newEncryptedStream(transportA, initSess.Send, initSess.Recv)
	encB := newEncryptedStream(transportB, respSess.Send, respSess.Recv)

	sendItem := &v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_Heartbeat{
			Heartbeat: &v1sync.SyncStreamItem_SyncActionHeartbeat{},
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := encA.Send(sendItem); err != nil {
			t.Errorf("send: %v", err)
		}
	}()

	recvItem, err := encB.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	wg.Wait()

	if recvItem.GetHeartbeat() == nil {
		t.Fatalf("expected heartbeat, got %T", recvItem.GetAction())
	}
}

func TestEncryptedStream_BidirectionalMultiMessage(t *testing.T) {
	initSess, respSess := runHandshake(t)

	transportA, transportB := newFakeStreamPair()
	encA := newEncryptedStream(transportA, initSess.Send, initSess.Recv)
	encB := newEncryptedStream(transportB, respSess.Send, respSess.Recv)

	heartbeat := &v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_Heartbeat{
			Heartbeat: &v1sync.SyncStreamItem_SyncActionHeartbeat{},
		},
	}

	var wg sync.WaitGroup

	// A→B direction
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			if err := encA.Send(heartbeat); err != nil {
				t.Errorf("A send %d: %v", i, err)
			}
		}
	}()
	for i := 0; i < 5; i++ {
		if _, err := encB.Receive(); err != nil {
			t.Fatalf("B receive %d: %v", i, err)
		}
	}
	wg.Wait()

	// B→A direction
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			if err := encB.Send(heartbeat); err != nil {
				t.Errorf("B send %d: %v", i, err)
			}
		}
	}()
	for i := 0; i < 5; i++ {
		if _, err := encA.Receive(); err != nil {
			t.Fatalf("A receive %d: %v", i, err)
		}
	}
	wg.Wait()
}

func TestEstablishEncryption_Integration(t *testing.T) {
	testutil.InstallZapLogger(t)
	transportA, transportB := newFakeStreamPair()

	var encA, encB syncCommandStreamTrait
	var transcriptA, transcriptB []byte
	var errA, errB error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		// A is the initiator (client side).
		encA, transcriptA, errA = establishEncryption(transportA, true)
	}()
	go func() {
		defer wg.Done()
		// B is the responder (server side).
		encB, transcriptB, errB = establishEncryption(transportB, false)
	}()
	wg.Wait()

	if errA != nil {
		t.Fatalf("establish A: %v", errA)
	}
	if errB != nil {
		t.Fatalf("establish B: %v", errB)
	}
	if len(transcriptA) == 0 || len(transcriptB) == 0 {
		t.Fatal("transcripts must be non-empty after handshake")
	}
	if string(transcriptA) != string(transcriptB) {
		t.Fatal("paired peers must agree on transport transcript")
	}

	heartbeat := &v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_Heartbeat{
			Heartbeat: &v1sync.SyncStreamItem_SyncActionHeartbeat{},
		},
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := encA.Send(heartbeat); err != nil {
			t.Errorf("send: %v", err)
		}
	}()

	recv, err := encB.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	wg.Wait()

	if recv.GetHeartbeat() == nil {
		t.Fatalf("expected heartbeat, got %T", recv.GetAction())
	}
}

func TestEstablishEncryption_ProtocolVersionMismatch(t *testing.T) {
	testutil.InstallZapLogger(t)
	transportA, transportB := newFakeStreamPair()

	// Responder receives a handshake with the wrong protocol version and
	// must reject it with a protocol error.
	junkPub := make([]byte, 32)
	if _, err := rand.Read(junkPub); err != nil {
		t.Fatal(err)
	}
	go func() {
		_ = transportA.Send(&v1sync.SyncStreamItem{
			Action: &v1sync.SyncStreamItem_EstablishSharedSecret{
				EstablishSharedSecret: &v1sync.SyncStreamItem_SyncEstablishSharedSecret{
					ProtocolVersion: cryptoutil.TransportProtocolVersion + 1,
					KemPublicKey:    junkPub,
				},
			},
		})
	}()

	if _, _, err := establishEncryption(transportB, false); err == nil {
		t.Fatal("expected protocol version mismatch to fail handshake")
	}
}
