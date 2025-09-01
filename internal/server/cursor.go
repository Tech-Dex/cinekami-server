package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash"
	"math"
)

type CursorSigner struct {
	key []byte
	h   func() hash.Hash
}

func NewCursorSigner(key []byte) *CursorSigner {
	return &CursorSigner{key: append([]byte(nil), key...), h: sha256.New}
}

// Active movies cursor: popularity(float64) + id(int64)
func (c *CursorSigner) EncodeMovies(popularity float64, id int64) string {
	payload := make([]byte, 16)
	binary.BigEndian.PutUint64(payload[0:8], math.Float64bits(popularity))
	binary.BigEndian.PutUint64(payload[8:16], uint64(id))
	mac := hmac.New(c.h, c.key)
	mac.Write(payload)
	sig := mac.Sum(nil)
	buf := append(payload, sig...)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func (c *CursorSigner) DecodeMovies(token string) (float64, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, 0, err
	}
	if len(raw) < 16+32 {
		return 0, 0, errors.New("invalid_cursor_length")
	}
	payload := raw[:16]
	sig := raw[16:]
	mac := hmac.New(c.h, c.key)
	mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return 0, 0, errors.New("invalid_cursor_signature")
	}
	pop := math.Float64frombits(binary.BigEndian.Uint64(payload[0:8]))
	id := int64(binary.BigEndian.Uint64(payload[8:16]))
	return pop, id, nil
}

// Tallies cursor: count(int64) + categoryLen(uint16) + category bytes
func (c *CursorSigner) EncodeTallies(count int64, category string) string {
	catBytes := []byte(category)
	payload := make([]byte, 8+2+len(catBytes))
	binary.BigEndian.PutUint64(payload[0:8], uint64(count))
	binary.BigEndian.PutUint16(payload[8:10], uint16(len(catBytes)))
	copy(payload[10:], catBytes)
	mac := hmac.New(c.h, c.key)
	mac.Write(payload)
	sig := mac.Sum(nil)
	buf := append(payload, sig...)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func (c *CursorSigner) DecodeTallies(token string) (int64, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, "", err
	}
	if len(raw) < 10+32 {
		return 0, "", errors.New("invalid_cursor_length")
	}
	payload := raw[:len(raw)-32]
	sig := raw[len(raw)-32:]
	mac := hmac.New(c.h, c.key)
	mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return 0, "", errors.New("invalid_cursor_signature")
	}
	cnt := int64(binary.BigEndian.Uint64(payload[0:8]))
	catLen := int(binary.BigEndian.Uint16(payload[8:10]))
	if 10+catLen != len(payload) {
		return 0, "", errors.New("invalid_cursor_payload")
	}
	category := string(payload[10:])
	return cnt, category, nil
}

// Snapshots cursor: movie_id(int64)
func (c *CursorSigner) EncodeSnapshots(movieID int64) string {
	payload := make([]byte, 8)
	binary.BigEndian.PutUint64(payload, uint64(movieID))
	mac := hmac.New(c.h, c.key)
	mac.Write(payload)
	sig := mac.Sum(nil)
	buf := append(payload, sig...)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func (c *CursorSigner) DecodeSnapshots(token string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	if len(raw) < 8+32 {
		return 0, errors.New("invalid_cursor_length")
	}
	payload := raw[:8]
	sig := raw[8:]
	mac := hmac.New(c.h, c.key)
	mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return 0, errors.New("invalid_cursor_signature")
	}
	id := int64(binary.BigEndian.Uint64(payload))
	return id, nil
}
