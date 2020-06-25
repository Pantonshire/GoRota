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

import (
    "time"
)

type Atom struct {
    Epoch int
    Time  uint
}

type TimeSystem interface {
    //Returns the length of a single epoch in atoms.
    EpochLength() int
    //Returns the number of epochs from y to x (epoch(x) - epoch(y)).
    EpochDelta(x time.Time, y time.Time) int
    //Returns the time x encoded as an atomic time and a number of atoms from the fixpoint.
    EncodeTime(fixpoint time.Time, x time.Time) Atom
    //Returns the time decoded from an atomic time and a number of atoms from the fixpoint.
    DecodeTime(fixpoint time.Time, at Atom) time.Time
}

func (t Atom) Mod(ts TimeSystem) Atom {
    epoch := t.Epoch
    atomTime := int(t.Time)
    epochLength := ts.EpochLength()
    for ; atomTime >= epochLength; atomTime -= epochLength {
        epoch++
    }
    return Atom{Epoch: epoch, Time: uint(atomTime)}
}

func (t Atom) Clamp(ts TimeSystem) Atom {
    epochLength := ts.EpochLength()
    if int(t.Time) >= epochLength {
        t.Time = uint(epochLength-1)
    }
    return t
}
