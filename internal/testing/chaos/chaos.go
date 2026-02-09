// Package chaos provides utilities for chaos testing parsers with corrupt inputs.
//
// Chaos testing intentionally corrupts valid inputs to verify parsers handle
// malformed data gracefully without panicking.
package chaos

import (
	"math/rand"
	"unicode/utf8"
)

// Corruptor defines methods for corrupting input data.
type Corruptor struct {
	rng *rand.Rand
}

// NewCorruptor creates a new Corruptor with the given seed.
func NewCorruptor(seed int64) *Corruptor {
	return &Corruptor{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// Mutation represents a type of corruption applied to input.
type Mutation int

const (
	ByteFlip Mutation = iota
	ByteDelete
	ByteInsert
	ByteReplace
	Utf8Corrupt
	Truncation
	BitInversion
)

// Corrupt applies a random corruption to the input.
func (c *Corruptor) Corrupt(input []byte) []byte {
	if len(input) == 0 {
		return c.insertRandomBytes(nil)
	}

	mutation := Mutation(c.rng.Intn(7))
	switch mutation {
	case ByteFlip:
		return c.byteFlip(input)
	case ByteDelete:
		return c.byteDelete(input)
	case ByteInsert:
		return c.byteInsert(input)
	case ByteReplace:
		return c.byteReplace(input)
	case Utf8Corrupt:
		return c.utf8Corrupt(input)
	case Truncation:
		return c.truncate(input)
	case BitInversion:
		return c.bitInversion(input)
	default:
		return input
	}
}

// CorruptN applies n random corruptions to the input.
func (c *Corruptor) CorruptN(input []byte, n int) []byte {
	result := make([]byte, len(input))
	copy(result, input)

	for i := 0; i < n; i++ {
		result = c.Corrupt(result)
	}

	return result
}

// byteFlip flips random bits in random bytes.
func (c *Corruptor) byteFlip(input []byte) []byte {
	result := make([]byte, len(input))
	copy(result, input)

	if len(result) == 0 {
		return result
	}

	// Flip 1-3 random bytes
	n := c.rng.Intn(3) + 1
	for i := 0; i < n; i++ {
		idx := c.rng.Intn(len(result))
		result[idx] ^= byte(1 << c.rng.Intn(8))
	}

	return result
}

// byteDelete removes random bytes.
func (c *Corruptor) byteDelete(input []byte) []byte {
	if len(input) <= 1 {
		return input
	}

	idx := c.rng.Intn(len(input))
	return append(input[:idx], input[idx+1:]...)
}

// byteInsert inserts random bytes at random positions.
func (c *Corruptor) byteInsert(input []byte) []byte {
	idx := c.rng.Intn(len(input) + 1)
	b := byte(c.rng.Intn(256))
	return append(input[:idx], append([]byte{b}, input[idx:]...)...)
}

// byteReplace replaces a byte with a random byte.
func (c *Corruptor) byteReplace(input []byte) []byte {
	result := make([]byte, len(input))
	copy(result, input)

	if len(result) == 0 {
		return result
	}

	idx := c.rng.Intn(len(result))
	result[idx] = byte(c.rng.Intn(256))
	return result
}

// utf8Corrupt corrupts UTF-8 sequences.
func (c *Corruptor) utf8Corrupt(input []byte) []byte {
	result := make([]byte, len(input))
	copy(result, input)

	// Find UTF-8 multi-byte sequences and corrupt them
	for i := 0; i < len(result); {
		r, size := utf8.DecodeRune(result[i:])
		if r == utf8.RuneError && size > 1 {
			// Found an invalid UTF-8 sequence, corrupt it more
			if c.rng.Float64() < 0.5 {
				result[i] = byte(c.rng.Intn(256))
			}
		}
		i += size
	}

	// Also inject some invalid UTF-8 bytes
	if len(result) > 0 && c.rng.Float64() < 0.3 {
		idx := c.rng.Intn(len(result))
		// Insert invalid UTF-8 start byte
		result[idx] = 0xC0 | byte(c.rng.Intn(0x20))
	}

	return result
}

// truncate randomly truncates the input.
func (c *Corruptor) truncate(input []byte) []byte {
	if len(input) <= 1 {
		return input
	}

	// Truncate at random position
	pos := c.rng.Intn(len(input)-1) + 1
	return input[:pos]
}

// bitInversion inverts random bits in the input.
func (c *Corruptor) bitInversion(input []byte) []byte {
	result := make([]byte, len(input))
	copy(result, input)

	if len(result) == 0 {
		return result
	}

	// Invert 1-5 bits
	n := c.rng.Intn(5) + 1
	for i := 0; i < n; i++ {
		idx := c.rng.Intn(len(result))
		bit := c.rng.Intn(8)
		result[idx] ^= (1 << bit)
	}

	return result
}

// insertRandomBytes inserts random bytes.
func (c *Corruptor) insertRandomBytes(input []byte) []byte {
	n := c.rng.Intn(10) + 1
	bytes := make([]byte, n)
	c.rng.Read(bytes)
	return append(input, bytes...)
}

// GenerateCorpus generates a corpus of corrupted inputs from a valid input.
func (c *Corruptor) GenerateCorpus(valid []byte, count int) [][]byte {
	corpus := make([][]byte, count)
	for i := 0; i < count; i++ {
		// Vary the corruption intensity
		intensity := c.rng.Intn(5) + 1
		corpus[i] = c.CorruptN(valid, intensity)
	}
	return corpus
}
