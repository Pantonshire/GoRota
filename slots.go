/*
 * GoRota: a service scheduling library for Go
 * Copyright (C) 2020 Thomas Panton
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package gorota

import "errors"

const maxRunLength = 1 << 7

type Slots struct {
    Bytes []byte
}

type SlotsPatch struct {
    Start int
    Patch Slots
}

var (
    ErrNoTime        = errors.New("no temporal data given")
    ErrDiscontinuity = errors.New("temporal discontinuity in given intervals")
)

func NewSlots(bytes []byte) Slots {
    return Slots{Bytes: bytes}
}

func NewSlotsPatch(start int, bytes []byte) SlotsPatch {
    return SlotsPatch{Start: start, Patch: NewSlots(bytes)}
}

func decodeAvailable(b byte) bool {
    return b & 0b10000000 != 0
}

func decodeRunLength(b byte) int {
    return int((b & 0b01111111) + 1)
}

func encodeRun(available bool, runLength int) byte {
    l := byte(runLength - 1)
    if available {
        return l | 0b10000000
    } else {
        return l & 0b01111111
    }
}

func IntervalsToSlots(intervals []BoolInterval, ts TimeSystem) (Slots, error) {
    bytes, err := IntervalsToBytes(intervals, true, true, ts)
    if err != nil {
        return Slots{}, err
    }
    return NewSlots(bytes), nil
}

func IntervalsToSlotsPatch(intervals []BoolInterval, ts TimeSystem) (SlotsPatch, error) {
    if len(intervals) == 0 {
        return SlotsPatch{}, ErrNoTime
    }
    bytes, err := IntervalsToBytes(intervals, false, false, ts)
    if err != nil {
        return SlotsPatch{}, err
    }
    return NewSlotsPatch(intervals[0].From, bytes), nil
}

func IntervalsToBytes(intervals []BoolInterval, padHead bool, padTail bool, ts TimeSystem) ([]byte, error) {
    if len(intervals) == 0 {
        return nil, ErrNoTime
    }

    start := intervals[0].From
    end := intervals[len(intervals)-1].Until

    var bytesRequired int
    var intervalGaps = make([]int, len(intervals))
    lastIntervalUntil := start

    for i, interval := range intervals {
        if interval.From < lastIntervalUntil {
            return nil, ErrDiscontinuity
        }
        if i > 0 {
            gapLength := interval.From - lastIntervalUntil
            intervalGaps[i-1] = gapLength
            bytesRequired += (gapLength + maxRunLength - 1) / maxRunLength
        }
        bytesRequired += (interval.Length() + maxRunLength - 1) / maxRunLength
        lastIntervalUntil = interval.Until
    }

    if bytesRequired == 0 {
        return nil, ErrNoTime
    }

    headLength := 0
    if padHead {
        headLength = start
        if headLength > 0 {
            bytesRequired += (headLength + maxRunLength - 1) / maxRunLength
        }
    }

    tailLength := 0
    if padTail {
        tailLength = ts.EpochLength() - end
        if tailLength > 0 {
            bytesRequired += (tailLength + maxRunLength - 1) / maxRunLength
        }
    }

    bytes := make([]byte, bytesRequired)
    i := 0

    for ; headLength > 0; headLength -= maxRunLength {
        runLength := headLength
        if runLength > maxRunLength {
            runLength = maxRunLength
        }
        bytes[i] = encodeRun(false, runLength)
        i++
    }

    for j, interval := range intervals {
        for intervalLength := interval.Length(); intervalLength > 0; intervalLength -= maxRunLength {
            runLength := intervalLength
            if runLength > maxRunLength {
                runLength = maxRunLength
            }
            bytes[i] = encodeRun(interval.Value, runLength)
            i++
        }

        for gapLength := intervalGaps[j]; gapLength > 0; gapLength -= maxRunLength {
            runLength := gapLength
            if runLength > maxRunLength {
                runLength = maxRunLength
            }
            bytes[i] = encodeRun(false, runLength)
            i++
        }
    }

    for ; tailLength > 0; tailLength -= maxRunLength {
        runLength := tailLength
        if runLength > maxRunLength {
            runLength = maxRunLength
        }
        bytes[i] = encodeRun(false, runLength)
        i++
    }

    return bytes, nil
}

func (slots Slots) ToIntervals(startTime int) []BoolInterval {
    if len(slots.Bytes) == 0 {
        return []BoolInterval{}
    }

    var intervals []BoolInterval
    intervalStart := 0
    currentValue := false
    t := startTime

    for i, b := range slots.Bytes {
        available := decodeAvailable(b)
        runLength := decodeRunLength(b)
        if available != currentValue {
            if i > 0 {
                intervals = append(intervals, NewBoolInterval(intervalStart, t, currentValue))
                intervalStart = t
            }
            currentValue = available
        }
        t += runLength
    }

    intervals = append(intervals, NewBoolInterval(intervalStart, t, currentValue))

    return intervals
}

func (slots Slots) ApplyPatch(patch SlotsPatch) Slots {
    if len(patch.Patch.Bytes) == 0 {
        return slots
    }

    var patchedBytes []byte

    i := 0
    t := 0

    for ; i < len(slots.Bytes); i++ {
        runLength := decodeRunLength(slots.Bytes[i])
        if t+runLength > patch.Start {
            break
        }
        patchedBytes = append(patchedBytes, slots.Bytes[i])
        t += runLength
    }

    if i < len(slots.Bytes) {
        if head := patch.Start - t; head > 0 {
            patchedBytes = append(patchedBytes, encodeRun(decodeAvailable(slots.Bytes[i]), head))
        }

        t = patch.Start

        for j := 0; j < len(patch.Patch.Bytes); j++ {
            patchedBytes = append(patchedBytes, patch.Patch.Bytes[j])
            t += decodeRunLength(patch.Patch.Bytes[j])
        }

        k := 0
        for i = 0; i < len(slots.Bytes); i++ {
            runLength := decodeRunLength(slots.Bytes[i])
            if k+runLength > t {
                break
            }
            k += runLength
        }

        if i < len(slots.Bytes) {
            if tail := decodeRunLength(slots.Bytes[i]) + k - t; tail > 0 {
                patchedBytes = append(patchedBytes, encodeRun(decodeAvailable(slots.Bytes[i]), tail))
            }

            i++
            for ; i < len(slots.Bytes); i++ {
                patchedBytes = append(patchedBytes, slots.Bytes[i])
            }
        }
    }

    return NewSlots(patchedBytes)
}

func (slots Slots) ApplyPatches(patches []SlotsPatch) Slots {
    patched := slots
    for _, patch := range patches {
        patched = slots.ApplyPatch(patch)
    }
    return patched
}

func (slots Slots) Intersection(other Slots) Slots {
    var intersection []byte

    i, j := 0, 0
    t := 0
    si, sj := 0, 0
    r := 0

    if len(slots.Bytes) > 0 && len(other.Bytes) > 0 {
        li, lj := decodeRunLength(slots.Bytes[0]), decodeRunLength(other.Bytes[0])
        vi, vj := decodeAvailable(slots.Bytes[0]), decodeAvailable(other.Bytes[0])
        vr := vi && vj

        for i < len(slots.Bytes) && j < len(other.Bytes) {
            if (vi && vj) != vr || r == maxRunLength {
                intersection = append(intersection, encodeRun(vr, r))
                vr = vi && vj
                r = 0
            }

            t++
            r++

            if t == si + li {
                i++
                si = t
                if i < len(slots.Bytes) {
                    li = decodeRunLength(slots.Bytes[i])
                    vi = decodeAvailable(slots.Bytes[i])
                }
            }

            if t == sj + lj {
                j++
                sj = t
                if j < len(other.Bytes) {
                    lj = decodeRunLength(other.Bytes[j])
                    vj = decodeAvailable(other.Bytes[j])
                }
            }
        }

        intersection = append(intersection, encodeRun(vr, r))
    }

    return NewSlots(intersection)
}

func (slots Slots) IsAvailable(interval Interval) bool {
    if interval.Until <= interval.From {
        return false
    }

    i := 0
    t := 0

    for i < len(slots.Bytes) {
        runLength := decodeRunLength(slots.Bytes[i])
        if t+runLength > interval.From {
            break
        }
        t += runLength
        i++
    }

    for i < len(slots.Bytes) && t < interval.Until {
        available := decodeAvailable(slots.Bytes[i])
        runLength := decodeRunLength(slots.Bytes[i])
        if !available {
            return false
        }
        t += runLength
        i++
    }

    return t >= interval.Until
}

func (slots Slots) AvailableIntervals(length int, between Interval) []Interval {
    if len(slots.Bytes) == 0 {
        return []Interval{}
    }

    i := 0
    t := 0

    for ; i < len(slots.Bytes); i++ {
        runLength := decodeRunLength(slots.Bytes[i])
        if t+runLength > between.From {
            break
        }
        t += runLength
    }

    var intervals []Interval

    if i < len(slots.Bytes) {
        chaining := false
        blockStart := between.From

        for ; i < len(slots.Bytes) && t < between.Until; i++ {
            available := decodeAvailable(slots.Bytes[i])
            runLength := decodeRunLength(slots.Bytes[i])

            if available && !chaining {
                blockStart = t
                chaining = true
            } else if !available && chaining {
                blockEnd := t
                if blockEnd > between.Until {
                    blockEnd = between.Until
                }

                for j := blockStart; j+length <= blockEnd; j++ {
                    interval := NewInterval(j, j+length)
                    intervals = append(intervals, interval)
                }

                chaining = false
            }

            t += runLength
        }

        if chaining {
            blockEnd := t
            if blockEnd > between.Until {
                blockEnd = between.Until
            }

            for j := blockStart; j+length <= blockEnd; j++ {
                interval := NewInterval(j, j+length)
                intervals = append(intervals, interval)
            }
        }
    }

    return intervals
}
