package main

import "time"

type duration time.Duration

func (d *duration) Set(value string) error {
	x, err := time.ParseDuration(value)
	*d = duration(x)
	return err
}

func (d duration) String() string {
	return time.Duration(d).String()
}
