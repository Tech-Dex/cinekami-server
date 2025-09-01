package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash"
	"math"
)

// Codec lists the signer methods the handlers rely on.
// Implementations must be safe for concurrent use.
type Codec interface {
	EncodeMoviesCursor(popularity float64, id int64) string
	DecodeMoviesCursor(token string) (float64, int64, error)

	EncodeTalliesCursor(count int64, category string) string
	DecodeTalliesCursor(token string) (int64, string, error)

	EncodeSnapshotsCursor(movieID int64) string
	DecodeSnapshotsCursor(token string) (int64, error)
}

// HMAC implements Codec using HMAC-SHA256 for integrity.
// It encodes payloads as base64 URL without padding.
type HMAC struct {
	key []byte
	h   func() hash.Hash
}

// NewHMAC creates an HMAC signer with the provided secret key.
func NewHMAC(key []byte) *HMAC {
	return &HMAC{key: append([]byte(nil), key...), h: sha256.New}
}

// seal signs the payload and returns a base64url token payload||sig.
func (c *HMAC) seal(payload []byte) string {
	mac := hmac.New(c.h, c.key)
	mac.Write(payload)
	sig := mac.Sum(nil)
	buf := append(payload, sig...)
	return base64.RawURLEncoding.EncodeToString(buf)
}

// open verifies the token and returns the payload bytes.
func (c *HMAC) open(token string, minPayloadLen int) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	if len(raw) < minPayloadLen+32 {
		return nil, errors.New("invalid_cursor_length")
	}
	payload := raw[:len(raw)-32]
	sig := raw[len(raw)-32:]
	mac := hmac.New(c.h, c.key)
	mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return nil, errors.New("invalid_cursor_signature")
	}
	return payload, nil
}

// Active movies signer: popularity(float64) + id(int64)
func (c *HMAC) EncodeMoviesCursor(popularity float64, id int64) string {
	payload := make([]byte, 16)
	binary.BigEndian.PutUint64(payload[0:8], math.Float64bits(popularity))
	binary.BigEndian.PutUint64(payload[8:16], uint64(id))
	return c.seal(payload)
}

func (c *HMAC) DecodeMoviesCursor(token string) (float64, int64, error) {
	payload, err := c.open(token, 16)
	if err != nil {
		return 0, 0, err
	}
	pop := math.Float64frombits(binary.BigEndian.Uint64(payload[0:8]))
	id := int64(binary.BigEndian.Uint64(payload[8:16]))
	return pop, id, nil
}

// Tallies signer: count(int64) + categoryLen(uint16) + category bytes
func (c *HMAC) EncodeTalliesCursor(count int64, category string) string {
	catBytes := []byte(category)
	payload := make([]byte, 8+2+len(catBytes))
	binary.BigEndian.PutUint64(payload[0:8], uint64(count))
	binary.BigEndian.PutUint16(payload[8:10], uint16(len(catBytes)))
	copy(payload[10:], catBytes)
	return c.seal(payload)
}

func (c *HMAC) DecodeTalliesCursor(token string) (int64, string, error) {
	payload, err := c.open(token, 10)
	if err != nil {
		return 0, "", err
	}
	cnt := int64(binary.BigEndian.Uint64(payload[0:8]))
	catLen := int(binary.BigEndian.Uint16(payload[8:10]))
	if 10+catLen != len(payload) {
		return 0, "", errors.New("invalid_cursor_payload")
	}
	category := string(payload[10:])
	return cnt, category, nil
}

// Snapshots signer: movie_id(int64)
func (c *HMAC) EncodeSnapshotsCursor(movieID int64) string {
	payload := make([]byte, 8)
	binary.BigEndian.PutUint64(payload, uint64(movieID))
	return c.seal(payload)
}

func (c *HMAC) DecodeSnapshotsCursor(token string) (int64, error) {
	payload, err := c.open(token, 8)
	if err != nil {
		return 0, err
	}
	id := int64(binary.BigEndian.Uint64(payload))
	return id, nil
}
