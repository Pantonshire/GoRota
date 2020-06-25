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
    "math"
    "time"
)

const (
    weekHours  = 168
    weekLength = time.Hour * weekHours
)

type WeekSystem struct {
    AtomDuration time.Duration
}

func (ws WeekSystem) EpochLength() int {
    return int(weekLength.Nanoseconds() / ws.AtomDuration.Nanoseconds())
}

func (ws WeekSystem) EpochDelta(x time.Time, y time.Time) int {
    return roundedWeeksBetween(startOfWeek(x), startOfWeek(y))
}

func (ws WeekSystem) EncodeTime(fixpoint time.Time, x time.Time) Atom {
    xWeekStart := startOfWeek(x)
    return Atom{
        Epoch: roundedWeeksBetween(xWeekStart, startOfWeek(fixpoint)),
        Time:  uint(x.Sub(xWeekStart).Nanoseconds() / ws.AtomDuration.Nanoseconds()),
    }.Clamp(ws)
}

func (ws WeekSystem) DecodeTime(fixpoint time.Time, at Atom) time.Time {
    at = at.Mod(ws)
    decodedTime := startOfWeek(fixpoint).AddDate(0, 0, at.Epoch*7)
    atomTimeDuration := ws.AtomDuration * time.Duration(at.Time)
    days := int(atomTimeDuration.Hours() / 24)
    if days > 0 {
        decodedTime = decodedTime.AddDate(0, 0, days)
        atomTimeDuration -= time.Hour * time.Duration(days*24)
    }
    return decodedTime.Add(atomTimeDuration)
}

func startOfWeek(t time.Time) time.Time {
    year, month, day := t.AddDate(0, 0, -mod(int(t.Weekday())-1, 7)).Date()
    return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func mod(x int, base int) int {
    return (x%base + base) % base
}

func roundedWeeksBetween(x time.Time, y time.Time) int {
    return int(math.Round(x.Sub(y).Hours() / weekHours))
}
