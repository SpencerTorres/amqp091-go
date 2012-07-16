// Copyright (c) 2012, Sean Treadway, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Source code and contact info at http://github.com/streadway/amqp

package amqp

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"testing"
	"time"
)

func integrationUri(t *testing.T) (*URI, bool) {
	urlStr := os.Getenv("AMQP_URL")
	if urlStr == "" {
		t.Logf("Skipping; AMQP_URL not found in the environment")
		return nil, false
	}

	uri, err := ParseURI(urlStr)
	if err != nil {
		t.Errorf("Failed to parse integration URI: %s", err)
		return nil, false
	}

	return &uri, true
}

// Returns a conneciton to the AMQP if the AMQP_URL environment
// variable is set and a connnection can be established.
func integrationConnection(t *testing.T, name string) *Connection {
	if uri, ok := integrationUri(t); ok {
		conn, err := Dial(uri.String())
		if err != nil {
			t.Errorf("Failed to connect to integration server: %s", err)
			return nil
		}

		return conn
	}

	return nil
}

func assertConsumeBody(t *testing.T, messages chan Delivery, body []byte) bool {
	select {
	case msg := <-messages:
		if bytes.Compare(msg.Body, body) != 0 {
			t.Errorf("Message body does not match have: %v expect %v", msg.Body, body)
			return false
		}
		return true
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Timeout waiting for %s", body)
		return false
	}
	panic("unreachable")
}

// Pulls out the CRC and verifies the remaining content against the CRC
func assertMessageCrc32(t *testing.T, msg []byte, assert string) {
	size := binary.BigEndian.Uint32(msg[:4])

	crc := crc32.NewIEEE()
	crc.Write(msg[8:])

	if binary.BigEndian.Uint32(msg[4:8]) != crc.Sum32() {
		t.Fatalf("Message does not match CRC: %s", assert)
	}

	if int(size) != len(msg)-8 {
		t.Fatalf("Message does not match size, should=%d, is=%d: %s", size, len(msg)-8, assert)
	}
}

// Creates a random body size with a leading 32-bit CRC in network byte order
// that verifies the remaining slice
func generateCrc32Random(size int) []byte {
	msg := make([]byte, size+8)
	if _, err := io.ReadFull(rand.Reader, msg); err != nil {
		panic(err)
	}

	crc := crc32.NewIEEE()
	crc.Write(msg[8:])

	binary.BigEndian.PutUint32(msg[0:4], uint32(size))
	binary.BigEndian.PutUint32(msg[4:8], crc.Sum32())

	return msg
}