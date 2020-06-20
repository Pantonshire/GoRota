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
    "errors"
    "fmt"
)

type Interval struct {
    //The first time included in this interval (inclusive)
    From  int `json:"from"`
    //The last time included in this interval (inclusive)
    Until int `json:"until"`
}

type BoolInterval struct {
    Time  Interval `json:"time"`
    Value bool     `json:"value"`
}

var ErrBadTimeInterval = errors.New("negative or zero time interval")

func NewInterval(from int, until int) Interval {
    return Interval{From: from, Until: until}
}

func NewBoolInterval(from int, until int, value bool) BoolInterval {
    return BoolInterval{Time: NewInterval(from, until), Value: value}
}

func (iv Interval) Validate() error {
    if iv.Until < iv.From {
        return ErrBadTimeInterval
    }
    return nil
}

func (iv Interval) Length() int {
    return (iv.Until - iv.From) + 1
}

func (iv Interval) String() string {
    return fmt.Sprintf("[%d,%d]", iv.From, iv.Until)
}

func (bi BoolInterval) String() string {
    return fmt.Sprintf("%s=%t", bi.Time, bi.Value)
}
