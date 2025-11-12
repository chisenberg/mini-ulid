package miniulid

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

// ID represents the compact 40-bit identifier.
type ID uint64

const (
	daysBits    = 15
	minutesBits = 11
	randomBits  = 14

	randomMask  = (1 << randomBits) - 1
	minutesMask = (1 << minutesBits) - 1
	daysMask    = (1 << daysBits) - 1

	totalBits = daysBits + minutesBits + randomBits
	totalSize = 8
)

const encodeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var (
	epoch          = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	errTimePast    = fmt.Errorf("miniulid: time before %s", epoch.Format(time.RFC3339))
	errTimeFuture  = fmt.Errorf("miniulid: time beyond supported range")
	errInvalidChar = fmt.Errorf("miniulid: invalid Crockford character")
	errLength      = fmt.Errorf("miniulid: encoded form must be %d characters", totalSize)
)

var decodeAlphabet = func() map[byte]uint8 {
	m := make(map[byte]uint8, len(encodeAlphabet)*2)
	for i, r := range encodeAlphabet {
		c := byte(r)
		m[c] = uint8(i)
		if c >= 'A' && c <= 'Z' {
			m[c+32] = uint8(i) // lowercase letters
		}
	}
	alt := map[byte]uint8{
		'i': 1, 'I': 1, 'l': 1, 'L': 1,
		'o': 0, 'O': 0,
	}
	for k, v := range alt {
		m[k] = uint8(v)
	}
	return m
}()

// Generate produces a new ID using the current UTC time and crypto/rand entropy.
func Generate() (ID, error) {
	return GenerateWithTime(time.Now().UTC(), rand.Reader)
}

// MustGenerate is a convenience helper that panics on error.
func MustGenerate() ID {
	id, err := Generate()
	if err != nil {
		panic(err)
	}
	return id
}

// GenerateWithTime produces an ID for the provided time, using the supplied entropy reader for the random bits.
func GenerateWithTime(t time.Time, entropy io.Reader) (ID, error) {
	dayCount, minuteOfDay, err := splitTime(t)
	if err != nil {
		return 0, err
	}

	randomValue, err := random14(entropy)
	if err != nil {
		return 0, fmt.Errorf("miniulid: random entropy: %w", err)
	}

	value := (uint64(dayCount) << (minutesBits + randomBits)) |
		(uint64(minuteOfDay) << randomBits) |
		uint64(randomValue)

	return ID(value), nil
}

// GenerateWithComponents builds an ID from a timestamp and a user-supplied counter/random value.
func GenerateWithComponents(t time.Time, random uint16) (ID, error) {
	if random > randomMask {
		return 0, fmt.Errorf("miniulid: random value overflow (max %d)", randomMask)
	}

	dayCount, minuteOfDay, err := splitTime(t)
	if err != nil {
		return 0, err
	}

	value := (uint64(dayCount) << (minutesBits + randomBits)) |
		(uint64(minuteOfDay) << randomBits) |
		uint64(random)

	return ID(value), nil
}

// Parse decodes an encoded string into an ID.
func Parse(encoded string) (ID, error) {
	if len(encoded) != totalSize {
		return 0, errLength
	}

	var value uint64
	for _, r := range encoded {
		c := byte(r)
		v, ok := decodeAlphabet[c]
		if !ok {
			return 0, fmt.Errorf("%w: %q", errInvalidChar, c)
		}
		value = (value << 5) | uint64(v)
	}

	return ID(value), nil
}

// FromInt64 converts a 40-bit integer representation into an ID.
func FromInt64(v int64) (ID, error) {
	if v < 0 {
		return 0, fmt.Errorf("miniulid: negative value")
	}
	if v>>totalBits != 0 {
		return 0, fmt.Errorf("miniulid: value exceeds %d bits", totalBits)
	}
	return ID(v), nil
}

// Int64 returns the 40-bit integer representation.
func (id ID) Int64() int64 {
	return int64(id)
}

// String returns the Crockford Base32 encoded form.
func (id ID) String() string {
	var buf [totalSize]byte
	value := uint64(id)

	for i := totalSize - 1; i >= 0; i-- {
		buf[i] = encodeAlphabet[int(value&31)]
		value >>= 5
	}

	return string(buf[:])
}

// Time reconstructs the original minute-precision UTC time.
func (id ID) Time() time.Time {
	value := uint64(id)

	random := uint16(value & randomMask)
	_ = random // ensures we keep the variable for clarity; random not used directly
	value >>= randomBits

	minuteOfDay := uint16(value & minutesMask)
	value >>= minutesBits

	days := uint16(value & daysMask)

	t := epoch.AddDate(0, 0, int(days))
	return t.Add(time.Duration(minuteOfDay) * time.Minute)
}

// Components returns the day, minute, and random segments for inspection.
func (id ID) Components() (days uint16, minuteOfDay uint16, random uint16) {
	value := uint64(id)

	random = uint16(value & randomMask)
	value >>= randomBits

	minuteOfDay = uint16(value & minutesMask)
	value >>= minutesBits

	days = uint16(value & daysMask)
	return
}

func splitTime(t time.Time) (uint16, uint16, error) {
	utc := t.UTC()
	if utc.Before(epoch) {
		return 0, 0, errTimePast
	}

	duration := utc.Sub(epoch)
	days := duration / (24 * time.Hour)
	if days >= 1<<daysBits {
		return 0, 0, errTimeFuture
	}

	minuteOfDay := utc.Hour()*60 + utc.Minute()
	return uint16(days), uint16(minuteOfDay), nil
}

func random14(entropy io.Reader) (uint16, error) {
	var buffer [2]byte
	if _, err := io.ReadFull(entropy, buffer[:]); err != nil {
		return 0, err
	}
	return uint16(buffer[0])<<8 | uint16(buffer[1])&randomMask, nil
}
