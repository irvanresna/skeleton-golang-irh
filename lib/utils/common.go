package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"
)

// Contains find is string in slices
func Contains(slices []string, comparizon string) bool {
	for _, a := range slices {
		if a == comparizon {
			return true
		}
	}

	return false
}

// ArrayContains(Array, Any) find is number in slice
func ArrayContains(slice interface{}, need interface{}) bool {
	arr := reflect.ValueOf(slice)
	switch arr.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < arr.Len(); i++ {
			if reflect.DeepEqual(need, arr.Index(i).Interface()) {
				return true
			}
		}
	case reflect.Map:
		for _, m := range arr.MapKeys() {
			if reflect.DeepEqual(need, arr.MapIndex(m).Interface()) {
				return true
			}
		}
	}

	return false
}

// IsEmail check email the real email string
func IsEmail(email string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	return re.MatchString(email)
}

// IsNumeric ...
func IsNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// ToInt ...
func ToInt(s string) (int, error) {
	if len(s) == 0 {
		return 0, nil
	}

	if !IsNumeric(s) {
		return 0, fmt.Errorf("%s", "must be a number")
	}

	return strconv.Atoi(s)
}

//GetTimeLocationWIB get WIB location
func GetTimeLocationWIB() *time.Location {
	wib, _ := time.LoadLocation("Asia/Jakarta")
	return wib
}

// ArrayUnique : for same data type, not available for struct or map
func ArrayUnique(arr interface{}) interface{} {
	val := reflect.ValueOf(arr)

	var kind reflect.Kind
	index := map[interface{}]int{}
	for i := 0; i < val.Len(); i++ {
		v := val.Index(i)
		kind = v.Kind()
		index[v.Interface()] = 0
	}

	var result interface{}
	if len(index) > 0 {
		switch kind {
		case reflect.Int:
			vint := make([]int, 0)
			for k, _ := range index {
				vint = append(vint, k.(int))
			}
			result = vint
		case reflect.Int32:
			vint := make([]int32, 0)
			for k, _ := range index {
				vint = append(vint, k.(int32))
			}
			result = vint
		case reflect.Int8:
			vint := make([]int8, 0)
			for k, _ := range index {
				vint = append(vint, k.(int8))
			}
			result = vint
		case reflect.Int16:
			vint := make([]int16, 0)
			for k, _ := range index {
				vint = append(vint, k.(int16))
			}
			result = vint
		case reflect.Int64:
			vint := make([]int64, 0)
			for k, _ := range index {
				vint = append(vint, k.(int64))
			}
			result = vint
		case reflect.Uint:
			vuint := make([]uint, 0)
			for k, _ := range index {
				vuint = append(vuint, k.(uint))
			}
			result = vuint
		case reflect.Uint8:
			vuint := make([]uint8, 0)
			for k, _ := range index {
				vuint = append(vuint, k.(uint8))
			}
			result = vuint
		case reflect.Uint16:
			vuint := make([]uint16, 0)
			for k, _ := range index {
				vuint = append(vuint, k.(uint16))
			}
			result = vuint
		case reflect.Uint32:
			vuint := make([]uint32, 0)
			for k, _ := range index {
				vuint = append(vuint, k.(uint32))
			}
			result = vuint
		case reflect.Uint64:
			vuint := make([]uint64, 0)
			for k, _ := range index {
				vuint = append(vuint, k.(uint64))
			}
			result = vuint
		case reflect.Float32:
			vfloat := make([]float32, 0)
			for k, _ := range index {
				vfloat = append(vfloat, k.(float32))
			}
			result = vfloat
		case reflect.Float64:
			vfloat := make([]float64, 0)
			for k, _ := range index {
				vfloat = append(vfloat, k.(float64))
			}
			result = vfloat
		case reflect.String:
			vstring := make([]string, 0)
			for k, _ := range index {
				vstring = append(vstring, k.(string))
			}
			result = vstring
		}
	}

	return result
}
