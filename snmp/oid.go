package snmp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Oid []int

//字符串转换
func (o Oid) String() string {
	if len(o) == 0 {
		return "."
	}
	var result string
	for _, val := range o {
		result += fmt.Sprintf(".%d", val)
	}
	return result
}

func MustParseOid(o string) Oid {
	result, err := ParseOid(o)
	if err != nil {
		panic(err)
	}
	return result
}

//解析OID
func ParseOid(oid string) (Oid, error) {
	// Special case "." = [], "" = []
	if oid == "." || oid == "" {
		return Oid{}, nil
	}
	if oid[0] == '.' {
		oid = oid[1:]
	}
	oidParts := strings.Split(oid, ".")
	res := make([]int, len(oidParts))
	for idx, val := range oidParts {
		parsedVal, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		res[idx] = parsedVal
	}
	result := Oid(res)

	return result, nil
}

func DecodeOid(raw []byte) (*Oid, error) {
	if len(raw) < 1 {
		return nil, errors.New("0 byte oid doesn't exist")
	}

	result := make([]int, 2)
	result[0] = int(raw[0] / 40)
	result[1] = int(raw[0] % 40)
	val := 0
	for idx, b := range raw {
		if idx == 0 {
			continue
		}
		if b < 128 {
			val = val*128 + int(b)
			result = append(result, val)
			val = 0
		} else {
			val = val*128 + int(b%128)
		}
	}
	r := Oid(result)
	return &r, nil
}

func (o Oid) Encode() ([]byte, error) {
	if len(o) < 2 {
		return nil, errors.New("oid needs to be at least 2 long")
	}
	var result []byte
	start := (40 * o[0]) + o[1]
	result = append(result, byte(start))
	for i := 2; i < len(o); i++ {
		val := o[i]

		var toadd []int
		if val == 0 {
			toadd = append(toadd, 0)
		}
		for val > 0 {
			toadd = append(toadd, val%128)
			val /= 128
		}

		for i := len(toadd) - 1; i >= 0; i-- {
			sevenbits := toadd[i]
			if i != 0 {
				result = append(result, 128+byte(sevenbits))
			} else {
				result = append(result, byte(sevenbits))
			}
		}
	}
	return result, nil
}

func (o Oid) Copy() Oid {
	dest := make([]int, len(o))
	copy(dest, o)
	return Oid(dest)
}

func (o Oid) Within(other Oid) bool {
	if len(other) > len(o) {
		return false
	}
	for idx, val := range other {
		if o[idx] != val {
			return false
		}
	}
	return true
}
