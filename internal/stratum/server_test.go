package stratum

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestServer_StartStop(t *testing.T) {
	srv := NewServer(1.0, testLogger())
	err := srv.Start("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer srv.Stop()

	if srv.SessionCount() != 0 {
		t.Error("should have 0 sessions initially")
	}
}

func TestServer_MinerConnection(t *testing.T) {
	srv := NewServer(1.0, testLogger())
	err := srv.Start("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer srv.Stop()

	addr := srv.listener.Addr().String()

	// Connect a mock miner
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Subscribe
	subscribe := `{"id":1,"method":"mining.subscribe","params":["test/1.0"]}` + "\n"
	conn.Write([]byte(subscribe))

	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read subscribe response: %v", err)
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("unmarshal subscribe response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("subscribe returned error: %v", resp.Error)
	}

	// The result should be an array with subscriptions, extranonce1, extranonce2_size
	resultBytes, _ := json.Marshal(resp.Result)
	var result []interface{}
	json.Unmarshal(resultBytes, &result)
	if len(result) != 3 {
		t.Fatalf("subscribe result should have 3 elements, got %d", len(result))
	}

	// Drain the mining.set_difficulty notification sent after subscribe
	reader.ReadBytes('\n')

	// Authorize
	authorize := `{"id":2,"method":"mining.authorize","params":["testworker","x"]}` + "\n"
	conn.Write([]byte(authorize))

	line, err = reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read authorize response: %v", err)
	}

	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("unmarshal authorize response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("authorize returned error: %v", resp.Error)
	}

	// Verify session count
	time.Sleep(50 * time.Millisecond)
	if srv.SessionCount() != 1 {
		t.Errorf("session count = %d, want 1", srv.SessionCount())
	}
}

func TestServer_ExtranonceUniqueness(t *testing.T) {
	srv := NewServer(1.0, testLogger())
	err := srv.Start("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer srv.Stop()

	addr := srv.listener.Addr().String()

	extranonces := make(map[string]bool)

	for i := 0; i < 5; i++ {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			t.Fatalf("connect %d failed: %v", i, err)
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)

		subscribe := fmt.Sprintf(`{"id":%d,"method":"mining.subscribe","params":["test/1.0"]}`, i+1) + "\n"
		conn.Write([]byte(subscribe))

		line, _ := reader.ReadBytes('\n')
		var resp Response
		json.Unmarshal(line, &resp)

		resultBytes, _ := json.Marshal(resp.Result)
		var result []interface{}
		json.Unmarshal(resultBytes, &result)

		en1, ok := result[1].(string)
		if !ok {
			t.Fatalf("extranonce1 not a string")
		}

		if extranonces[en1] {
			t.Errorf("duplicate extranonce1: %s", en1)
		}
		extranonces[en1] = true
	}
}

func TestVardiff(t *testing.T) {
	v := NewVardiff(1.0)
	if v.Difficulty() != 1.0 {
		t.Errorf("initial difficulty = %f, want 1.0", v.Difficulty())
	}
}

func TestServer_BroadcastJob(t *testing.T) {
	srv := NewServer(1.0, testLogger())
	err := srv.Start("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer srv.Stop()

	addr := srv.listener.Addr().String()

	// Connect and authorize a miner
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	conn.Write([]byte(`{"id":1,"method":"mining.subscribe","params":["test"]}` + "\n"))
	reader.ReadBytes('\n') // subscribe response
	reader.ReadBytes('\n') // mining.set_difficulty notification

	conn.Write([]byte(`{"id":2,"method":"mining.authorize","params":["worker","x"]}` + "\n"))
	reader.ReadBytes('\n') // authorize response

	time.Sleep(50 * time.Millisecond)

	// Broadcast a job
	job := &Job{
		ID:             "1",
		PrevHash:       "0000000000000000000000000000000000000000000000000000000000000000",
		Coinbase1:      "01000000",
		Coinbase2:      "ffffffff",
		MerkleBranches: []string{},
		Version:        "20000000",
		NBits:          "1d00ffff",
		NTime:          "65432100",
		CleanJobs:      true,
	}

	srv.BroadcastJob(job)

	// Read the notification
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read job notification: %v", err)
	}

	var notif Notification
	if err := json.Unmarshal(line, &notif); err != nil {
		t.Fatalf("unmarshal notification: %v", err)
	}

	if notif.Method != "mining.notify" {
		t.Errorf("notification method = %s, want mining.notify", notif.Method)
	}
}
