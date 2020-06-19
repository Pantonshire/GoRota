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

type Interval struct {
    //The first time included in this interval (inclusive)
    From  int `json:"from"`
    //The last time included in this interval (inclusive)
    Until int `json:"until"`
}

var ErrBadTimeInterval = errors.New("negative or zero time interval")

func NewInterval(from int, until int) Interval {
    return Interval{From: from, Until: until}
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
