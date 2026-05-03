package syncapi

import (
	"sync"
	"testing"

	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
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

func TestEncryptedStream_RoundTrip(t *testing.T) {
	alice, err := cryptoutil.GenerateECDHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := cryptoutil.GenerateECDHKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	gcm, err := cryptoutil.DeriveSessionKey(alice.Private, bob.Public)
	if err != nil {
		t.Fatal(err)
	}

	aliceIsSmaller := string(alice.Public.Bytes()) < string(bob.Public.Bytes())

	transportA, transportB := newFakeStreamPair()
	encA := newEncryptedStream(transportA, gcm, aliceIsSmaller)
	encB := newEncryptedStream(transportB, gcm, !aliceIsSmaller)

	// Send a heartbeat from A to B
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
	alice, _ := cryptoutil.GenerateECDHKeyPair()
	bob, _ := cryptoutil.GenerateECDHKeyPair()
	gcm, _ := cryptoutil.DeriveSessionKey(alice.Private, bob.Public)

	aliceIsSmaller := string(alice.Public.Bytes()) < string(bob.Public.Bytes())

	transportA, transportB := newFakeStreamPair()
	encA := newEncryptedStream(transportA, gcm, aliceIsSmaller)
	encB := newEncryptedStream(transportB, gcm, !aliceIsSmaller)

	heartbeat := &v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_Heartbeat{
			Heartbeat: &v1sync.SyncStreamItem_SyncActionHeartbeat{},
		},
	}

	// Send 5 messages A→B sequentially, then 5 messages B→A sequentially
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
	transportA, transportB := newFakeStreamPair()

	var encA, encB syncCommandStreamTrait
	var errA, errB error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		encA, errA = establishEncryption(transportA)
	}()
	go func() {
		defer wg.Done()
		encB, errB = establishEncryption(transportB)
	}()
	wg.Wait()

	if errA != nil {
		t.Fatalf("establish A: %v", errA)
	}
	if errB != nil {
		t.Fatalf("establish B: %v", errB)
	}

	// Verify encrypted communication works
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
