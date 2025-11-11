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

## Encoding Process (Golang Example)

```go
// combine fields into 40-bit value
id := (uint64(daysSince2020) << (11 + 14)) |
      (uint64(minuteOfDay) << 14) |
      uint64(random14)

// encode to Crockford Base32 (8 chars)
encoded := CrockfordEncode40(id)
```
