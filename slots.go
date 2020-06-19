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
    bytes []byte
}

type SlotsPatch struct {
    Start int
    Patch Slots
}

type Slot struct {
    Time  Interval `json:"time"`
    Value bool     `json:"value"`
}

var ErrDiscontinuity = errors.New("temporal discontinuity in given intervals")

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

func newSlots(intervals []Slot, periodLength int, padHead bool, padTail bool) ([]byte, error) {
    if len(intervals) == 0 {
        return nil, errors.New("no intervals given")
    }

    start := intervals[0].Time.From
    end := intervals[len(intervals)-1].Time.Until

    var bytesRequired int
    var intervalGaps = make([]int, len(intervals))
    lastIntervalUntil := start - 1

    for i, interval := range intervals {
        if interval.Time.From <= lastIntervalUntil {
            return nil, ErrDiscontinuity
        }
        if i > 0 {
            gapLength := interval.Time.From - (lastIntervalUntil + 1)
            intervalGaps[i-1] = gapLength
            bytesRequired += (gapLength + maxRunLength - 1) / maxRunLength
        }
        bytesRequired += (interval.Time.Length() + maxRunLength - 1) / maxRunLength
        lastIntervalUntil = interval.Time.Until
    }

    if bytesRequired == 0 {
        return nil, errors.New("no temporal progression in given intervals")
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
        tailLength = (periodLength - 1) - end
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
        for intervalLength := interval.Time.Length(); intervalLength > 0; intervalLength -= maxRunLength {
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

func (patch SlotsPatch) apply(slots Slots) Slots {
    if len(patch.Patch.bytes) == 0 {
        return slots
    }

    var patchedBytes []byte

    i := 0
    t := 0

    for ; i < len(slots.bytes); i++ {
        runLength := decodeRunLength(slots.bytes[i])
        if t+runLength > patch.Start {
            break
        }
        patchedBytes = append(patchedBytes, slots.bytes[i])
        t += runLength
    }

    if i < len(slots.bytes) {
        trimmedHeadAvailable := decodeAvailable(slots.bytes[i])
        trimmedHeadLength := patch.Start - t
        if trimmedHeadLength > 0 {
            patchedBytes = append(patchedBytes, encodeRun(trimmedHeadAvailable, trimmedHeadLength))
        }

        t = patch.Start

        for j := 0; j < len(patch.Patch.bytes); j++ {
            patchedBytes = append(patchedBytes, patch.Patch.bytes[j])
            k := t
            t += decodeRunLength(patch.Patch.bytes[j])

            for ; i < len(slots.bytes); i++ {
                runLength := decodeRunLength(slots.bytes[i])
                if k+runLength >= t {
                    break
                }
                k += runLength
            }
        }

        if i < len(slots.bytes) {
            k := 0
            for j := 0; j < i; j++ {
                k += decodeRunLength(slots.bytes[j])
            }

            trimmedTailAvailable := decodeAvailable(slots.bytes[i])
            trimmedTailLength := decodeRunLength(slots.bytes[i]) + k - t
            if trimmedTailLength > 0 {
                patchedBytes = append(patchedBytes, encodeRun(trimmedTailAvailable, trimmedTailLength))
            }

            i++
            for ; i < len(slots.bytes); i++ {
                patchedBytes = append(patchedBytes, slots.bytes[i])
            }
        }
    }

    return Slots{bytes: patchedBytes}
}
