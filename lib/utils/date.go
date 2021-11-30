package utils

import (
	"fmt"
	"time"
)

// ToUTCfromGMT7 ...
func ToUTCfromGMT7(strTime string) (time.Time, error) {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.Now(), err
	}

	date, err := time.ParseInLocation("2006-01-02 15:04:05", strTime, location)
	if err != nil {
		fmt.Printf("\nerror when parse strTime [%s] -> err: %v\n", strTime, err)
		return time.Now(), err
	}

	return date.In(time.UTC), nil
}

// FromUTCLocationToGMT7 ...
func FromUTCLocationToGMT7(date time.Time) (time.Time, error){
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.Now(), err
	}

	return date.In(location), nil
}

// FromGMT7LocationUTCMin7 ...
func FromGMT7LocationUTCMin7(date time.Time) (time.Time, error) {
	date = date.Add(time.Hour * -7)
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.Now(), err
	}

	date = date.In(location)

	return date.In(time.UTC), nil
}