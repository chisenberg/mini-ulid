# Compact 40-bit Crockford ID Specification

A 40-bit sortable, compact, and human-friendly identifier.

---

## Overview

This ID format is designed for systems with moderate insert frequency  
(e.g. users, companies, devices, events), and aims to be:

- Chronologically **sortable**  
- **Compact** (8 Crockford Base32 chars)  
- **Collision-safe** within each minute  
- **Readable** and non-sequential

Example (`2024-08-18T15:30Z`, random segment `0x04D2` → decimal 1234):

- Encoded string: `0F5VD3YH`
- Components:
  - Days since 2020-01-01 UTC: `1689`
  - Minute of day: `930` (15:30)
  - Random/counter: `1234`

---

## Bit Layout

| Bits (high→low) | Field | Bits | Range | Description |
|------------------|--------|------|--------|--------------|
| 39–25 | **DaysSince2020** | 15 | 0–32767 | Days since `2020-01-01 UTC` (≈ 90 years, valid until 2109-10-29) |
| 24–14 | **MinuteOfDay** | 11 | 0–1439 | Minute of the day (0 = 00:00, 1439 = 23:59) |
| 13–0  | **RandomOrCounter** | 14 | 0–16383 | Random or sequential number per minute (≈ 16K IDs/min) |

**Total: 40 bits = 8 Crockford Base32 characters**

---

## Structure

[ DaysSince2020 (15) | MinuteOfDay (11) | RandomOrCounter (14) ]

Binary: DDDDDDDDDDDDDDD MMMMMMMMMMM RRRRRRRRRRRRRR

---

## Go Usage

### Installation

```sh
go get github.com/chisenberg/mini-ulid
```

```go
package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/chisenberg/mini-ulid"
)

func main() {
	// Generate with random entropy (crypto/rand).
	id, err := miniulid.Generate()
	if err != nil {
		panic(err)
	}
	// e.g. generate: 0F5VD3YH 1234567890 2024-08-18 15:30:00 +0000 UTC
	fmt.Println("generate:", id.String(), id.Int64(), id.Time())

	// Generate with supplied counter/random segment.
	// accepts 0-16383
	counter := uint16(42)
	withCounter, err := miniulid.GenerateWithComponents(time.Now(), counter)
	if err != nil {
		panic(err)
	}
	fmt.Println("counter:", withCounter.String())
	// counter: 0F5VD3YH

	// Generate with deterministic time + entropy reader.
	// minute precision (seconds ignored)
	t := time.Date(2024, 8, 18, 15, 30, 0, 0, time.UTC)
	// any io.Reader; lower 14 bits used
	entropy := bytes.NewReader([]byte{0x12, 0x34})
	withEntropy, err := miniulid.GenerateWithTime(t, entropy)
	if err != nil {
		panic(err)
	}
	fmt.Println("withTime:", withEntropy.String(), withEntropy.Time())
	// withTime: 0F5VD3YH 2024-08-18 15:30:00 +0000 UTC

	// Panic-on-error helper.
	must := miniulid.MustGenerate()
	fmt.Println("must:", must.String())
	// must: 0F5VD3YH

	// Parse from Crockford Base32 string.
	// accepts 8-char Crockford Base32 (case-insensitive)
	parsed, err := miniulid.Parse(withEntropy.String())
	if err != nil {
		panic(err)
	}
	fmt.Println("parse:", parsed.Int64(), parsed.Time())
	// parse: 1234567890 2024-08-18 15:30:00 +0000 UTC

	// Convert to and from the 40-bit int64 encoding.
	// input must fit in 40 bits and be >= 0
	back, err := miniulid.FromInt64(parsed.Int64())
	if err != nil {
		panic(err)
	}
	fmt.Println("int64:", back.String(), back.Int64())
	// int64: 0F5VD3YH 1234567890

	// Examine bitfield components.
	days, minuteOfDay, random := back.Components()
	fmt.Printf("parts: days=%d minute=%d random=%d\n", days, minuteOfDay, random)
	// parts: days=1689 minute=930 random=1234
}
```
