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
