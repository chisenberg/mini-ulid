package miniulid

import (
	"fmt"
	"sync"
	"time"
)

// ID represents the compact 40-bit identifier.
type ID uint64

const (
	daysBits    = 15
	minutesBits = 11
	counterBits = 14

	counterMask = (1 << counterBits) - 1
	minutesMask = (1 << minutesBits) - 1
	daysMask    = (1 << daysBits) - 1

	totalBits = daysBits + minutesBits + counterBits
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

var defaultMinuteCounter = &minuteCounter{}

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

// Generate produces a new ID using the current UTC minute and a monotonic counter.
func Generate() (ID, error) {
	now := time.Now().UTC()
	counter, err := defaultMinuteCounter.next(now)
	if err != nil {
		return 0, err
	}
	return GenerateWithComponents(now, counter)
}

// MustGenerate is a convenience helper that panics on error.
func MustGenerate() ID {
	id, err := Generate()
	if err != nil {
		panic(err)
	}
	return id
}

// GenerateWithComponents builds an ID from a timestamp and a user-supplied counter value.
func GenerateWithComponents(t time.Time, counter uint16) (ID, error) {
	if counter > counterMask {
		return 0, fmt.Errorf("miniulid: counter value overflow (max %d)", counterMask)
	}

	dayCount, minuteOfDay, err := splitTime(t)
	if err != nil {
		return 0, err
	}

	value := (uint64(dayCount) << (minutesBits + counterBits)) |
		(uint64(minuteOfDay) << counterBits) |
		uint64(counter)

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

	counter := uint16(value & counterMask)
	_ = counter // ensures we keep the variable for clarity; counter not used directly
	value >>= counterBits

	minuteOfDay := uint16(value & minutesMask)
	value >>= minutesBits

	days := uint16(value & daysMask)

	t := epoch.AddDate(0, 0, int(days))
	return t.Add(time.Duration(minuteOfDay) * time.Minute)
}

// Components returns the day, minute, and rancdom segments for inspection.
func (id ID) Components() (days uint16, minuteOfDay uint16, counter uint16) {
	value := uint64(id)

	counter = uint16(value & counterMask)
	value >>= counterBits

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

type minuteCounter struct {
	mu     sync.Mutex
	minute time.Time
	value  uint16
}

func (mc *minuteCounter) next(t time.Time) (uint16, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	currentMinute := t.UTC().Truncate(time.Minute)

	if mc.minute.IsZero() || !mc.minute.Equal(currentMinute) {
		mc.minute = currentMinute
		mc.value = 0
		return 0, nil
	}

	if mc.value == counterMask {
		return 0, fmt.Errorf("miniulid: counter overflow for minute %s", currentMinute.Format(time.RFC3339))
	}

	mc.value++
	return mc.value, nil
}
