// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package test_vector_utils

import (
	"encoding/json"
	"fmt"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr/polynomial"

	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type ElementTriplet struct {
	key1        fr.Element
	key2        fr.Element
	key2Present bool
	value       fr.Element
	used        bool
}

func (t *ElementTriplet) CmpKey(o *ElementTriplet) int {
	if cmp1 := t.key1.Cmp(&o.key1); cmp1 != 0 {
		return cmp1
	}

	if t.key2Present {
		if o.key2Present {
			return t.key2.Cmp(&o.key2)
		}
		return 1
	} else {
		if o.key2Present {
			return -1
		}
		return 0
	}
}

var HashCache = make(map[string]*HashMap)

func GetHash(path string) (*HashMap, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if h, ok := HashCache[path]; ok {
		return h, nil
	}
	var bytes []byte
	if bytes, err = os.ReadFile(path); err == nil {
		var asMap map[string]interface{}
		if err = json.Unmarshal(bytes, &asMap); err != nil {
			return nil, err
		}

		res := make(HashMap, 0, len(asMap))

		for k, v := range asMap {
			var entry ElementTriplet
			if _, err = SetElement(&entry.value, v); err != nil {
				return nil, err
			}

			key := strings.Split(k, ",")

			switch len(key) {
			case 1:
				entry.key2Present = false
			case 2:
				entry.key2Present = true
				if _, err = SetElement(&entry.key2, key[1]); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("cannot parse %T as one or two field elements", v)
			}
			if _, err = SetElement(&entry.key1, key[0]); err != nil {
				return nil, err
			}

			res = append(res, &entry)
		}

		res.sort()

		HashCache[path] = &res

		return &res, nil

	} else {
		return nil, err
	}
}

type HashMap []*ElementTriplet

func (t *ElementTriplet) writeKey(sb *strings.Builder) {
	sb.WriteRune('"')
	sb.WriteString(t.key1.String())
	if t.key2Present {
		sb.WriteRune(',')
		sb.WriteString(t.key2.String())
	}
	sb.WriteRune('"')
}
func (m *HashMap) UnusedEntries() []interface{} {
	unused := make([]interface{}, 0)
	for _, v := range *m {
		if !v.used {
			var vInterface interface{}
			if v.key2Present {
				vInterface = []interface{}{ElementToInterface(&v.key1), ElementToInterface(&v.key2)}
			} else {
				vInterface = ElementToInterface(&v.key1)
			}
			unused = append(unused, vInterface)
		}
	}
	return unused
}

func (m *HashMap) sort() {
	sort.Slice(*m, func(i, j int) bool {
		return (*m)[i].CmpKey((*m)[j]) <= 0
	})
}

func (m *HashMap) find(toFind *ElementTriplet) fr.Element {
	i := sort.Search(len(*m), func(i int) bool { return (*m)[i].CmpKey(toFind) >= 0 })

	if i < len(*m) && (*m)[i].CmpKey(toFind) == 0 {
		(*m)[i].used = true
		return (*m)[i].value
	}
	var sb strings.Builder
	sb.WriteString("no value available for input ")
	toFind.writeKey(&sb)
	panic(sb.String())
}

func (m *HashMap) FindPair(x *fr.Element, y *fr.Element) fr.Element {

	toFind := ElementTriplet{
		key1:        *x,
		key2Present: y != nil,
	}

	if y != nil {
		toFind.key2 = *y
	}

	return m.find(&toFind)
}

type MapHashTranscript struct {
	HashMap         *HashMap
	stateValid      bool
	resultAvailable bool
	state           fr.Element
}

func (m *MapHashTranscript) Update(i ...interface{}) {
	if len(i) > 0 {
		for _, x := range i {

			var xElement fr.Element
			if _, err := xElement.SetInterface(x); err != nil {
				panic(err.Error())
			}
			if m.stateValid {
				m.state = m.HashMap.FindPair(&xElement, &m.state)
			} else {
				m.state = m.HashMap.FindPair(&xElement, nil)
			}

			m.stateValid = true
		}
	} else { //just hash the state itself
		if !m.stateValid {
			panic("nothing to hash")
		}
		m.state = m.HashMap.FindPair(&m.state, nil)
	}
	m.resultAvailable = true
}

func (m *MapHashTranscript) Next(i ...interface{}) fr.Element {

	if len(i) > 0 || !m.resultAvailable {
		m.Update(i...)
	}
	m.resultAvailable = false
	return m.state
}

func (m *MapHashTranscript) NextN(N int, i ...interface{}) []fr.Element {

	if len(i) > 0 {
		m.Update(i...)
	}

	res := make([]fr.Element, N)

	for n := range res {
		res[n] = m.Next()
	}

	return res
}
func SetElement(z *fr.Element, value interface{}) (*fr.Element, error) {

	// TODO: Put this in element.SetString?
	switch v := value.(type) {
	case string:

		if sep := strings.Split(v, "/"); len(sep) == 2 {
			var denom fr.Element
			if _, err := z.SetString(sep[0]); err != nil {
				return nil, err
			}
			if _, err := denom.SetString(sep[1]); err != nil {
				return nil, err
			}
			denom.Inverse(&denom)
			z.Mul(z, &denom)
			return z, nil
		}

	case float64:
		asInt := int64(v)
		if float64(asInt) != v {
			return nil, fmt.Errorf("cannot currently parse float")
		}
		z.SetInt64(asInt)
		return z, nil
	}

	return z.SetInterface(value)
}

func SliceToElementSlice(slice []interface{}) ([]fr.Element, error) {
	elementSlice := make([]fr.Element, len(slice))
	for i, v := range slice {
		if _, err := SetElement(&elementSlice[i], v); err != nil {
			return nil, err
		}
	}
	return elementSlice, nil
}

func SliceEquals(a []fr.Element, b []fr.Element) error {
	if len(a) != len(b) {
		return fmt.Errorf("length mismatch %d≠%d", len(a), len(b))
	}
	for i := range a {
		if !a[i].Equal(&b[i]) {
			return fmt.Errorf("at index %d: %s ≠ %s", i, a[i].String(), b[i].String())
		}
	}
	return nil
}

func SliceSliceEquals(a [][]fr.Element, b [][]fr.Element) error {
	if len(a) != len(b) {
		return fmt.Errorf("length mismatch %d≠%d", len(a), len(b))
	}
	for i := range a {
		if err := SliceEquals(a[i], b[i]); err != nil {
			return fmt.Errorf("at index %d: %w", i, err)
		}
	}
	return nil
}

func PolynomialSliceEquals(a []polynomial.Polynomial, b []polynomial.Polynomial) error {
	if len(a) != len(b) {
		return fmt.Errorf("length mismatch %d≠%d", len(a), len(b))
	}
	for i := range a {
		if err := SliceEquals(a[i], b[i]); err != nil {
			return fmt.Errorf("at index %d: %w", i, err)
		}
	}
	return nil
}

func ElementToInterface(x *fr.Element) interface{} {
	text := x.Text(10)
	if len(text) < 10 && !strings.Contains(text, "/") {
		if i, err := strconv.Atoi(text); err != nil {
			panic(err.Error())
		} else {
			return i
		}
	}
	return text
}

func ElementSliceToInterfaceSlice(x interface{}) []interface{} {
	if x == nil {
		return nil
	}

	X := reflect.ValueOf(x)

	res := make([]interface{}, X.Len())
	for i := range res {
		xI := X.Index(i).Interface().(fr.Element)
		res[i] = ElementToInterface(&xI)
	}
	return res
}

func ElementSliceSliceToInterfaceSliceSlice(x interface{}) [][]interface{} {
	if x == nil {
		return nil
		return nil
	}

	X := reflect.ValueOf(x)

	res := make([][]interface{}, X.Len())
	for i := range res {
		res[i] = ElementSliceToInterfaceSlice(X.Index(i).Interface())
	}

	return res
}
