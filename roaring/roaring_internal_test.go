// Copyright 2017 Pilosa Corp.
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

package roaring

import (
	"reflect"
	"testing"
)

func TestRunAppendInterval(t *testing.T) {
	a := container{}
	tests := []struct {
		base []interval32
		app  interval32
		exp  int
	}{
		{
			base: []interval32{},
			app:  interval32{start: 22, last: 25},
			exp:  4,
		},
		{
			base: []interval32{{start: 20, last: 23}},
			app:  interval32{start: 22, last: 25},
			exp:  2,
		},
		{
			base: []interval32{{start: 20, last: 23}},
			app:  interval32{start: 21, last: 22},
			exp:  0,
		},
		{
			base: []interval32{{start: 20, last: 23}},
			app:  interval32{start: 19, last: 25},
			exp:  2, // runAppendInterval explicitly does not support intervals whose start is < c.runs[-1].start
		},
	}

	for i, test := range tests {
		a.runs = test.base
		if n := a.runAppendInterval(test.app); n != test.exp {
			t.Fatalf("test #%v expected %v, but got %v", i, test.exp, n)
		}
	}

}

func TestInterval32RunLen(t *testing.T) {
	iv := interval32{start: 7, last: 9}
	if iv.runlen() != 3 {
		t.Fatalf("should be 3")
	}
	iv = interval32{start: 7, last: 7}
	if iv.runlen() != 1 {
		t.Fatalf("should be 1")
	}
}

func TestContainerRunAdd(t *testing.T) {
	c := container{runs: make([]interval32, 0)}
	tests := []struct {
		op  uint32
		exp []interval32
	}{
		{1, []interval32{{start: 1, last: 1}}},
		{2, []interval32{{start: 1, last: 2}}},
		{4, []interval32{{start: 1, last: 2}, {start: 4, last: 4}}},
		{3, []interval32{{start: 1, last: 4}}},
		{10, []interval32{{start: 1, last: 4}, {start: 10, last: 10}}},
		{7, []interval32{{start: 1, last: 4}, {start: 7, last: 7}, {start: 10, last: 10}}},
		{6, []interval32{{start: 1, last: 4}, {start: 6, last: 7}, {start: 10, last: 10}}},
		{0, []interval32{{start: 0, last: 4}, {start: 6, last: 7}, {start: 10, last: 10}}},
		{8, []interval32{{start: 0, last: 4}, {start: 6, last: 8}, {start: 10, last: 10}}},
	}
	for _, test := range tests {
		c.mapped = true
		ret := c.add(test.op)
		if !ret {
			t.Fatalf("result of adding new bit should be true: %v", c.runs)
		}
		if !reflect.DeepEqual(c.runs, test.exp) {
			t.Fatalf("Should have %v, but got %v after adding %v", test.exp, c.runs, test.op)
		}
		if c.mapped {
			t.Fatalf("container should not be mapped after adding bit %v", test.op)
		}
	}
}

func TestContainerRunAdd2(t *testing.T) {
	c := container{runs: make([]interval32, 0)}
	ret := c.add(0)
	if !ret {
		t.Fatalf("result of adding new bit should be true: %v", c.runs)
	}
	if !reflect.DeepEqual(c.runs, []interval32{{start: 0, last: 0}}) {
		t.Fatalf("should have 1 run of length 1, but have %v", c.runs)
	}
	ret = c.add(0)
	if ret {
		t.Fatalf("result of adding existing bit should be false: %v", c.runs)
	}
}

func TestRunCountRange(t *testing.T) {
	c := container{runs: make([]interval32, 0)}
	cnt := c.runCountRange(2, 9)
	if cnt != 0 {
		t.Fatalf("should get 0 from empty container, but got: %v", cnt)
	}
	c.add(5)
	c.add(6)
	c.add(7)

	cnt = c.runCountRange(2, 9)
	if cnt != 3 {
		t.Fatalf("should get 3 from interval within range, but got: %v", cnt)
	}

	c.add(8)
	c.add(9)
	c.add(10)
	c.add(11)

	cnt = c.runCountRange(6, 8)
	if cnt != 2 {
		t.Fatalf("should get 2 from range within interval, but got: %v", cnt)
	}

	cnt = c.runCountRange(3, 9)
	if cnt != 4 {
		t.Fatalf("should get 4 from range overlaps front of interval, but got: %v", cnt)
	}

	cnt = c.runCountRange(9, 14)
	if cnt != 3 {
		t.Fatalf("should get 3 from range overlaps back of interval, but got: %v", cnt)
	}

	c.add(17)
	c.add(18)
	c.add(19)

	cnt = c.runCountRange(1, 22)
	if cnt != 10 {
		t.Fatalf("should get 10 from multiple ranges in interval, but got: %v", cnt)
	}

	c.add(13)
	c.add(14)

	cnt = c.runCountRange(6, 18)
	if cnt != 9 {
		t.Fatalf("should get 9 from multiple ranges overlapping both sides, but got: %v", cnt)
	}
}

func TestRunContains(t *testing.T) {
	c := container{runs: make([]interval32, 0)}
	if c.runContains(5) {
		t.Fatalf("empty run container should not contain 5")
	}
	c.add(5)
	if !c.runContains(5) {
		t.Fatalf("run container with 5 should contain 5")
	}

	c.add(6)
	c.add(7)

	c.add(9)
	c.add(10)
	c.add(11)

	if !c.runContains(10) {
		t.Fatalf("run container with 10 in second run should contain 10")
	}
}

func TestBitmapCountRange(t *testing.T) {
	c := container{}
	tests := []struct {
		start  uint32
		end    uint32
		bitmap []uint64
		exp    int
	}{
		{start: 0, end: 1, bitmap: []uint64{1}, exp: 1},
		{start: 2, end: 7, bitmap: []uint64{0xFFFFFFFFFFFFFF18}, exp: 2},
		{start: 67, end: 68, bitmap: []uint64{0, 0x8}, exp: 1},
		{start: 1, end: 68, bitmap: []uint64{0x3, 0x8, 0xF}, exp: 2},
		{start: 1, end: 258, bitmap: []uint64{0xF, 0x8, 0xA, 0x4, 0xFFFFFFFFFFFFFFFF}, exp: 9},
		{start: 66, end: 71, bitmap: []uint64{0xF, 0xFFFFFFFFFFFFFF18}, exp: 2},
		{start: 63, end: 64, bitmap: []uint64{0x8000000000000000}, exp: 1},
	}

	for i, test := range tests {
		c.bitmap = test.bitmap
		if ret := c.bitmapCountRange(test.start, test.end); ret != test.exp {
			t.Fatalf("test #%v count of %v from %v to %v should be %v but got %v", i, test.bitmap, test.start, test.end, test.exp, ret)
		}
	}
}

func TestRunRemove(t *testing.T) {
	c := container{runs: []interval32{{start: 2, last: 10}, {start: 12, last: 13}, {start: 15, last: 16}}}
	tests := []struct {
		op     uint32
		exp    []interval32
		expRet bool
	}{
		{2, []interval32{{start: 3, last: 10}, {start: 12, last: 13}, {start: 15, last: 16}}, true},
		{10, []interval32{{start: 3, last: 9}, {start: 12, last: 13}, {start: 15, last: 16}}, true},
		{12, []interval32{{start: 3, last: 9}, {start: 13, last: 13}, {start: 15, last: 16}}, true},
		{13, []interval32{{start: 3, last: 9}, {start: 15, last: 16}}, true},
		{16, []interval32{{start: 3, last: 9}, {start: 15, last: 15}}, true},
		{6, []interval32{{start: 3, last: 5}, {start: 7, last: 9}, {start: 15, last: 15}}, true},
		{8, []interval32{{start: 3, last: 5}, {start: 7, last: 7}, {start: 9, last: 9}, {start: 15, last: 15}}, true},
		{8, []interval32{{start: 3, last: 5}, {start: 7, last: 7}, {start: 9, last: 9}, {start: 15, last: 15}}, false},
		{1, []interval32{{start: 3, last: 5}, {start: 7, last: 7}, {start: 9, last: 9}, {start: 15, last: 15}}, false},
		{44, []interval32{{start: 3, last: 5}, {start: 7, last: 7}, {start: 9, last: 9}, {start: 15, last: 15}}, false},
	}

	for _, test := range tests {
		c.mapped = true
		ret := c.remove(test.op)
		if ret != test.expRet || !reflect.DeepEqual(c.runs, test.exp) {
			t.Fatalf("Unexpected result removing %v from runs. Expected %v, got %v. Expected %v, got %v", test.op, test.expRet, ret, test.exp, c.runs)
		}
		if ret && c.mapped {
			t.Fatalf("container was not unmapped although bit %v was removed", test.op)
		}
		if !ret && !c.mapped {
			t.Fatalf("container was unmapped although bit %v was not removed", test.op)
		}
	}
}

func TestRunMax(t *testing.T) {
	c := container{runs: []interval32{{start: 2, last: 10}, {start: 12, last: 13}, {start: 15, last: 16}}}
	max := c.max()
	if max != 16 {
		t.Fatalf("max for %v should be 16", c.runs)
	}

	c = container{runs: []interval32{}}
	max = c.max()
	if max != 0 {
		t.Fatalf("max for %v should be 0", c.runs)
	}
}

func TestIntersectionCountArrayRun(t *testing.T) {
	a := &container{array: []uint32{1, 5, 10, 11, 12}}
	b := &container{runs: []interval32{{start: 2, last: 10}, {start: 12, last: 13}, {start: 15, last: 16}}}

	ret := intersectionCountArrayRun(a, b)
	if ret != 3 {
		t.Fatalf("count of %v with %v should be 3, but got %v", a.array, b.runs, ret)
	}
}

func TestIntersectionCountBitmapRun(t *testing.T) {
	a := &container{bitmap: []uint64{0x8000000000000000}}
	b := &container{runs: []interval32{{start: 63, last: 64}}}

	ret := intersectionCountBitmapRun(a, b)
	if ret != 1 {
		t.Fatalf("count of %v with %v should be 1, but got %v", a.bitmap, b.runs, ret)
	}

	a = &container{bitmap: []uint64{0xF0000001, 0xFF00000000000000, 0xFF000000000000F0, 0x0F0000}}
	b = &container{runs: []interval32{{start: 29, last: 31}, {start: 125, last: 134}, {start: 191, last: 197}, {start: 200, last: 300}}}

	ret = intersectionCountBitmapRun(a, b)
	if ret != 14 {
		t.Fatalf("count of %v with %v should be 14, but got %v", a.bitmap, b.runs, ret)
	}
}

func TestIntersectionCountRunRun(t *testing.T) {
	a := &container{}
	b := &container{}
	tests := []struct {
		aruns []interval32
		bruns []interval32
		exp   uint64
	}{
		{
			aruns: []interval32{},
			bruns: []interval32{{start: 3, last: 8}}, exp: 0},
		{
			aruns: []interval32{{start: 2, last: 10}},
			bruns: []interval32{{start: 3, last: 8}}, exp: 6},
		{
			aruns: []interval32{{start: 2, last: 10}},
			bruns: []interval32{{start: 1, last: 11}}, exp: 9},
		{
			aruns: []interval32{{start: 2, last: 10}},
			bruns: []interval32{{start: 0, last: 2}}, exp: 1},
		{
			aruns: []interval32{{start: 2, last: 10}},
			bruns: []interval32{{start: 1, last: 10}}, exp: 9},
		{
			aruns: []interval32{{start: 2, last: 10}},
			bruns: []interval32{{start: 5, last: 12}}, exp: 6},
		{
			aruns: []interval32{{start: 2, last: 10}},
			bruns: []interval32{{start: 10, last: 99}}, exp: 1},
		{
			aruns: []interval32{{start: 2, last: 10}, {start: 44, last: 99}},
			bruns: []interval32{{start: 12, last: 14}}, exp: 0},
		{
			aruns: []interval32{{start: 2, last: 10}, {start: 12, last: 13}},
			bruns: []interval32{{start: 2, last: 10}, {start: 12, last: 13}}, exp: 11},
		{
			aruns: []interval32{{start: 8, last: 12}, {start: 15, last: 19}},
			bruns: []interval32{{start: 9, last: 9}, {start: 11, last: 17}}, exp: 6},
	}
	for i, test := range tests {
		a.runs = test.aruns
		b.runs = test.bruns
		ret := intersectionCountRunRun(a, b)
		if ret != test.exp {
			t.Fatalf("test #%v failed intersecting %v with %v should be %v, but got %v", i, test.aruns, test.bruns, test.exp, ret)
		}
	}
}

func TestIntersectArrayRun(t *testing.T) {
	a := &container{}
	b := &container{}
	tests := []struct {
		array []uint32
		runs  []interval32
		exp   []uint32
	}{
		{
			array: []uint32{1, 4, 5, 7, 10, 11, 12},
			runs:  []interval32{{start: 5, last: 10}},
			exp:   []uint32{5, 7, 10},
		},
		{
			array: []uint32{},
			runs:  []interval32{{start: 5, last: 10}},
			exp:   []uint32(nil),
		},
		{
			array: []uint32{1, 4, 5, 7, 10, 11, 12},
			runs:  []interval32{},
			exp:   []uint32(nil),
		},
		{
			array: []uint32{0, 1, 4, 5, 7, 10, 11, 12},
			runs:  []interval32{{start: 0, last: 5}, {start: 7, last: 7}},
			exp:   []uint32{0, 1, 4, 5, 7},
		},
	}

	for i, test := range tests {
		a.array = test.array
		b.runs = test.runs
		ret := intersectArrayRun(a, b)
		if !reflect.DeepEqual(ret.array, test.exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, test.exp, ret.array)
		}
	}
}

func TestIntersectRunRun(t *testing.T) {
	a := &container{}
	b := &container{}
	tests := []struct {
		aruns []interval32
		bruns []interval32
		exp   []interval32
		expN  int
	}{
		{
			aruns: []interval32{},
			bruns: []interval32{{start: 5, last: 10}},
			exp:   []interval32(nil),
			expN:  0,
		},
		{
			aruns: []interval32{{start: 5, last: 12}},
			bruns: []interval32{{start: 5, last: 10}},
			exp:   []interval32{{start: 5, last: 10}},
			expN:  6,
		},
		{
			aruns: []interval32{{start: 1, last: 3}, {start: 5, last: 5}, {start: 7, last: 8}, {start: 9, last: 12}},
			bruns: []interval32{{start: 5, last: 10}},
			exp:   []interval32{{start: 5, last: 5}, {start: 7, last: 10}},
			expN:  5,
		},
	}
	for i, test := range tests {
		a.runs = test.aruns
		b.runs = test.bruns
		ret := intersectRunRun(a, b)
		if !reflect.DeepEqual(ret.runs, test.exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, test.exp, ret.runs)
		}
		if ret.n != test.expN {
			t.Fatalf("test #%v expected n to be %v, but got %v", i, test.expN, ret.n)
		}
	}

}

func TestIntersectBitmapRunBitmap(t *testing.T) {
	a := &container{bitmap: make([]uint64, bitmapN)}
	b := &container{}
	tests := []struct {
		bitmap []uint64
		runs   []interval32
		exp    []uint64
		expN   int
	}{
		{
			bitmap: []uint64{1},
			runs:   []interval32{{start: 0, last: 0}, {start: 2, last: 5}, {start: 62, last: 71}, {start: 77, last: 4096}},
			exp:    []uint64{1},
			expN:   1,
		},
		{
			bitmap: []uint64{0xFFFFFFFFFFFFFFFF},
			runs:   []interval32{{start: 1, last: 1}},
			exp:    []uint64{2},
			expN:   1,
		},
		{
			bitmap: []uint64{0xFFFFFFFFFFFFFFFF},
			runs:   []interval32{{start: 1, last: 1}, {start: 10, last: 12}, {start: 61, last: 77}},
			exp:    []uint64{0xe000000000001C02},
			expN:   7,
		},
		{
			bitmap: []uint64{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			runs:   []interval32{{start: 1, last: 1}, {start: 61, last: 77}},
			exp:    []uint64{0xE000000000000002, 0x00000000000003FFF},
			expN:   18,
		},
		{
			bitmap: []uint64{0xFFFFFFFFFFFFFFFF, 1, 1, 1, 0xA, 1, 1, 0, 1},
			runs:   []interval32{{start: 63, last: 10000}},
			exp:    []uint64{0x8000000000000000, 1, 1, 1, 0xA, 1, 1, 0, 1},
			expN:   9,
		},
	}
	for i, test := range tests {
		for i, v := range test.bitmap {
			a.bitmap[i] = v
		}
		b.runs = test.runs
		b.n = 4097 // ;)
		exp := make([]uint64, bitmapN)
		for i, v := range test.exp {
			exp[i] = v
		}
		ret := intersectBitmapRun(a, b)
		if ret.isArray() {
			ret.arrayToBitmap()
		}
		if !reflect.DeepEqual(ret.bitmap, exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, exp, ret.bitmap)
		}
		if ret.n != test.expN {
			t.Fatalf("test #%v expected n to be %v, but got %v", i, test.expN, ret.n)
		}
	}

}

func TestIntersectBitmapRunArray(t *testing.T) {
	a := &container{bitmap: make([]uint64, bitmapN)}
	b := &container{}
	tests := []struct {
		bitmap []uint64
		runs   []interval32
		exp    []uint32
		expN   int
	}{
		{
			bitmap: []uint64{1},
			runs:   []interval32{{start: 0, last: 0}, {start: 2, last: 5}, {start: 62, last: 71}, {start: 77, last: 4096}},
			exp:    []uint32{0},
			expN:   1,
		},
		{
			bitmap: []uint64{0xFFFFFFFFFFFFFFFF},
			runs:   []interval32{{start: 1, last: 1}},
			exp:    []uint32{1},
			expN:   1,
		},
		{
			bitmap: []uint64{0xFFFFFFFFFFFFFFFF},
			runs:   []interval32{{start: 1, last: 1}, {start: 10, last: 12}, {start: 61, last: 77}},
			exp:    []uint32{1, 10, 11, 12, 61, 62, 63},
			expN:   7,
		},
		{
			bitmap: []uint64{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			runs:   []interval32{{start: 1, last: 1}, {start: 61, last: 68}},
			exp:    []uint32{1, 61, 62, 63, 64, 65, 66, 67, 68},
			expN:   9,
		},
		{
			bitmap: []uint64{0xFFFFFFFFFFFFFFFF, 1, 1, 1, 0xA, 1, 1, 0, 1},
			runs:   []interval32{{start: 63, last: 10000}},
			exp:    []uint32{63, 64, 128, 192, 257, 259, 320, 384, 512},
			expN:   9,
		},
	}
	for i, test := range tests {
		for i, v := range test.bitmap {
			a.bitmap[i] = v
		}
		b.runs = test.runs
		ret := intersectBitmapRun(a, b)
		if !reflect.DeepEqual(ret.array, test.exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, test.exp, ret.array)
		}
		if ret.n != test.expN {
			t.Fatalf("test #%v expected n to be %v, but got %v", i, test.expN, ret.n)
		}
	}

}

func TestUnionRunRun(t *testing.T) {
	a := &container{}
	b := &container{}
	tests := []struct {
		aruns []interval32
		bruns []interval32
		exp   []interval32
	}{
		{
			aruns: []interval32{},
			bruns: []interval32{{start: 5, last: 10}},
			exp:   []interval32{{start: 5, last: 10}},
		},
		{
			aruns: []interval32{{start: 5, last: 12}},
			bruns: []interval32{{start: 5, last: 10}},
			exp:   []interval32{{start: 5, last: 12}},
		},
		{
			aruns: []interval32{{start: 1, last: 3}, {start: 5, last: 5}, {start: 7, last: 8}, {start: 9, last: 12}},
			bruns: []interval32{{start: 5, last: 10}},
			exp:   []interval32{{start: 1, last: 3}, {start: 5, last: 12}},
		},
		{
			aruns: []interval32{{start: 1, last: 3}, {start: 5, last: 5}, {start: 7, last: 8}, {start: 9, last: 12}},
			bruns: []interval32{{start: 2, last: 65535}},
			exp:   []interval32{{start: 1, last: 65535}},
		},
		{
			aruns: []interval32{{start: 2, last: 65535}},
			bruns: []interval32{{start: 1, last: 3}, {start: 5, last: 5}, {start: 7, last: 8}, {start: 9, last: 12}},
			exp:   []interval32{{start: 1, last: 65535}},
		},
		{
			aruns: []interval32{{start: 1, last: 3}, {start: 5, last: 5}, {start: 7, last: 8}, {start: 9, last: 12}},
			bruns: []interval32{{start: 0, last: 65535}},
			exp:   []interval32{{start: 0, last: 65535}},
		},
		{
			aruns: []interval32{{start: 0, last: 65535}},
			bruns: []interval32{{start: 1, last: 3}, {start: 5, last: 5}, {start: 7, last: 8}, {start: 9, last: 12}},
			exp:   []interval32{{start: 0, last: 65535}},
		},
		{
			aruns: []interval32{{start: 1, last: 3}, {start: 5, last: 5}, {start: 7, last: 9}, {start: 12, last: 22}},
			bruns: []interval32{{start: 2, last: 8}, {start: 16, last: 27}, {start: 33, last: 34}},
			exp:   []interval32{{start: 1, last: 9}, {start: 12, last: 27}, {start: 33, last: 34}},
		},
	}
	for i, test := range tests {
		a.runs = test.aruns
		b.runs = test.bruns
		ret := unionRunRun(a, b)
		if !reflect.DeepEqual(ret.runs, test.exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, test.exp, ret.runs)
		}
	}
}

func TestUnionArrayRun(t *testing.T) {
	a := &container{}
	b := &container{}
	tests := []struct {
		array []uint32
		runs  []interval32
		exp   []uint32
	}{
		{
			array: []uint32{1, 4, 5, 7, 10, 11, 12},
			runs:  []interval32{{start: 5, last: 10}},
			exp:   []uint32{1, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
		{
			array: []uint32{},
			runs:  []interval32{{start: 5, last: 10}},
			exp:   []uint32{5, 6, 7, 8, 9, 10},
		},
		{
			array: []uint32{1, 4, 5, 7, 10, 11, 12},
			runs:  []interval32{},
			exp:   []uint32{1, 4, 5, 7, 10, 11, 12},
		},
		{
			array: []uint32{0, 1, 4, 5, 7, 10, 11, 12},
			runs:  []interval32{{start: 0, last: 5}, {start: 7, last: 7}},
			exp:   []uint32{0, 1, 2, 3, 4, 5, 7, 10, 11, 12},
		},
	}

	for i, test := range tests {
		a.array = test.array
		b.runs = test.runs
		ret := unionArrayRun(a, b)
		if !reflect.DeepEqual(ret.array, test.exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, test.exp, ret.array)
		}
	}
}

func TestBitmapSetRange(t *testing.T) {
	c := &container{bitmap: make([]uint64, bitmapN)}
	tests := []struct {
		bitmap []uint64
		start  uint64
		last   uint64
		exp    []uint64
		expN   int
	}{
		{
			bitmap: []uint64{0x0000000000FFF900},
			start:  9,
			last:   10,
			exp:    []uint64{0x0000000000FFFF00},
			expN:   16,
		},
		{
			bitmap: []uint64{0xFF0, 0xFF, 0xFF},
			start:  60,
			last:   130,
			exp:    []uint64{0xF000000000000FF0, 0xFFFFFFFFFFFFFFFF, 0xFF},
			expN:   84,
		},
	}

	for i, test := range tests {
		for i, v := range test.bitmap {
			c.bitmap[i] = v
		}
		c.n = c.countRange(0, 65535)
		c.bitmapSetRange(test.start, test.last)
		if !reflect.DeepEqual(c.bitmap[:len(test.exp)], test.exp) {
			t.Fatalf("test %#v expected %x, got %x", i, test.exp, c.bitmap[:len(test.bitmap)])
		}
		if test.expN != c.n {
			t.Fatalf("test #%v expected n to be %v, but got %v", i, test.expN, c.n)
		}
	}
}

func TestArrayToBitmap(t *testing.T) {
	a := &container{}
	tests := []struct{
		array []uint32
		exp []uint64
	}{
		{
			array: []uint32{0, 1, 2, 3},
			exp: []uint64{0xF},
		},
	}

	for i, test := range tests {
		exp := make([]uint64, bitmapN)
		for i, v := range test.exp {
			exp[i] = v
		}

		a.array = test.array
		a.arrayToBitmap()
		if !reflect.DeepEqual(a.bitmap, exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, exp, a.bitmap)
		}
	}
}

func TestRunToBitmap(t *testing.T) {
	a := &container{}
	tests := []struct{
		runs []interval32
		exp []uint64
	}{
		{
	        runs: []interval32{{start: 0, last: 0}},
	        exp: []uint64{1},
	    },
		{
	        runs: []interval32{{start: 0, last: 4}},
	        exp: []uint64{31},
	    },
		{
	        runs: []interval32{{start: 2, last: 2}, {start:5, last:7}, {start:13, last:14}, {start:17, last:17}},
	        exp: []uint64{155876},
	    },
	    {
	        runs: []interval32{{start: 0, last: 3}, {start:60, last: 67}},
	        exp: []uint64{0xF00000000000000F, 0x000000000000000F},
	    },
	}

	for i, test := range tests {
		exp := make([]uint64, bitmapN)
		for i, v := range test.exp {
			exp[i] = v
		}

		a.runs = test.runs
		a.runToBitmap()
		if !reflect.DeepEqual(a.bitmap, exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, exp, a.bitmap)
		}
	}
}

func TestBitmapToRun(t *testing.T) {
	a := &container{}
	tests := []struct{
		bitmap []uint64
		exp []interval32
	}{
		{
	        // single-bit run
	        bitmap: []uint64{1},
	        exp: []interval32{{start: 0, last: 0}},
	    },
		{
	        // single multi-bit run in one word
	        bitmap: []uint64{31},
	        exp: []interval32{{start: 0, last: 4}},
	    },
		{
	        // multiple runs in one word
	        bitmap: []uint64{155876},
	        exp: []interval32{{start: 2, last: 2}, {start:5, last:7}, {start:13, last:14}, {start:17, last:17}},
	    },
	    {
	        // span two words, both mixed
	        bitmap: []uint64{0xF00000000000000F, 0x000000000000000F},
	        exp: []interval32{{start: 0, last: 3}, {start:60, last: 67}},
	    },
	    {
	        // span two words, first = maxBitmap
	        bitmap: []uint64{0xFFFFFFFFFFFFFFFF, 0xF},
	        exp: []interval32{{start: 0, last: 67}},
	    },
	    {
	        // span two words, second = maxBitmap
	        bitmap: []uint64{0xF000000000000000, 0xFFFFFFFFFFFFFFFF},
	        exp: []interval32{{start: 60, last: 127}},
	    },
	    {
	        // span three words
	        bitmap: []uint64{0xF000000000000000, 0xFFFFFFFFFFFFFFFF, 0xF},
	        exp: []interval32{{start: 60, last: 131}},
	    },
	}

	for i, test := range tests {
		a.bitmap = make([]uint64, bitmapN)
		n := 0
		for i, v := range test.bitmap {
			a.bitmap[i] = v
			n += int(popcount(v))
		}
		a.n = n
		a.bitmapToRun()
		if !reflect.DeepEqual(a.runs, test.exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, test.exp, a.runs)
		}
	}
}

func TestArrayToRun(t *testing.T) {
	a := &container{}
	tests := []struct{
		array []uint32
		exp []interval32
	}{
		{
	        array: []uint32{0},
	        exp: []interval32{{start: 0, last: 0}},
	    },
		{
	        array: []uint32{0, 1, 2, 3, 4},
	        exp: []interval32{{start: 0, last: 4}},
	    },
		{
	        array: []uint32{2, 5, 6, 7, 13, 14, 17},
	        exp: []interval32{{start: 2, last: 2}, {start:5, last:7}, {start:13, last:14}, {start:17, last:17}},
	    },
	}

	for i, test := range tests {
		a.array = test.array
		a.n = len(test.array)
		a.arrayToRun()
		if !reflect.DeepEqual(a.runs, test.exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, test.exp, a.runs)
		}
	}
}

func TestRunToArray(t *testing.T) {
	a := &container{}
	tests := []struct{
		runs []interval32
		exp []uint32
	}{
		{
	        runs: []interval32{{start: 0, last: 0}},
	        exp: []uint32{0},
	    },
		{
	        runs: []interval32{{start: 0, last: 4}},
	        exp: []uint32{0, 1, 2, 3, 4},
	    },
		{
	        runs: []interval32{{start: 2, last: 2}, {start:5, last:7}, {start:13, last:14}, {start:17, last:17}},
	        exp: []uint32{2, 5, 6, 7, 13, 14, 17},
	    },
	}

	for i, test := range tests {
		a.runs = test.runs
		a.n = len(test.exp)
		a.runToArray()
		if !reflect.DeepEqual(a.array, test.exp) {
			t.Fatalf("test #%v expected %v, but got %v", i, test.exp, a.array)
    }
  }
}

func TestBitmapZeroRange(t *testing.T) {
	c := &container{bitmap: make([]uint64, bitmapN)}
	tests := []struct {
		bitmap []uint64
		start  uint64
		last   uint64
		exp    []uint64
		expN   int
	}{
		{
			bitmap: []uint64{0x0000000000FFFF00},
			start:  9,
			last:   10,
			exp:    []uint64{0x0000000000FFF900},
			expN:   14,
		},
		{
			bitmap: []uint64{0xFF0, 0xFF, 0xFF},
			start:  60,
			last:   130,
			exp:    []uint64{0xFF0, 0, 0xF8},
			expN:   13,
		},
	}

	for i, test := range tests {
		for i, v := range test.bitmap {
			c.bitmap[i] = v
		}
		c.n = c.countRange(0, 65535)
		c.bitmapZeroRange(test.start, test.last)
		if !reflect.DeepEqual(c.bitmap[:len(test.exp)], test.exp) {
			t.Fatalf("test %#v expected %x, got %x", i, test.exp, c.bitmap[:len(test.bitmap)])
		}
		if test.expN != c.n {
			t.Fatalf("test #%v expected n to be %v, but got %v", i, test.expN, c.n)
		}
		for i, _ := range test.bitmap {
			c.bitmap[i] = 0
		}
	}

}

func TestUnionBitmapRun(t *testing.T) {
	a := &container{bitmap: make([]uint64, bitmapN)}
	b := &container{}
	tests := []struct {
		bitmap []uint64
		runs   []interval32
		exp    []uint64
		expN   int
	}{
		{
			bitmap: []uint64{2},
			runs:   []interval32{{start: 0, last: 0}, {start: 2, last: 5}, {start: 62, last: 71}, {start: 77, last: 78}},
			exp:    []uint64{0xC00000000000003F, 0x60FF},
			expN:   18,
		},
	}
	for i, test := range tests {
		for i, v := range test.bitmap {
			a.bitmap[i] = v
		}
		a.n = a.bitmapCountRange(0, 65535)
		b.runs = test.runs
		ret := unionBitmapRun(a, b)
		if ret.isArray() {
			ret.arrayToBitmap()
		}
		if !reflect.DeepEqual(ret.bitmap[:len(test.exp)], test.exp) {
			t.Fatalf("test #%v expected %x, but got %x", i, test.exp, ret.bitmap[:len(test.exp)])
		}
		if ret.n != test.expN {
			t.Fatalf("test #%v expected n to be %v, but got %v", i, test.expN, ret.n)
		}
		for i, _ := range test.bitmap {
			a.bitmap[i] = 0
		}
	}

}

func TestBitmapCountRuns(t *testing.T) {
	c := &container{bitmap: make([]uint64, bitmapN)}
	tests := []struct {
		bitmap []uint64
		exp    int
	}{
		{
			bitmap: []uint64{0xFF00FF00},
			exp:    2,
		},
		{
			bitmap: []uint64{0xFF00FF0000000000, 0x1},
			exp:    2,
		},
		{
			bitmap: []uint64{0xFF00FF0000000000, 0x2, 0x100},
			exp:    4,
		},
		{
			bitmap: []uint64{0xFF00FF0000000000, 0x1010101FF0101010, 0x100},
			exp:    10,
		},
	}

	for i, test := range tests {
		for j, v := range test.bitmap {
			c.bitmap[j] = v
		}

		ret := c.bitmapCountRuns()
		if ret != test.exp {
			t.Fatalf("test #%v expected %v but got %v", i, test.exp, ret)
		}

		for j, _ := range test.bitmap {
			c.bitmap[j] = 0
		}
	}

	test := tests[3]
	for j, v := range test.bitmap {
		c.bitmap[1024-len(test.bitmap)+j] = v

	}
	ret := c.bitmapCountRuns()
	if ret != test.exp {
		t.Fatalf("test at end expected %v but got %v", test.exp, ret)
	}
}

func TestArrayCountRuns(t *testing.T) {
	c := &container{}
	tests := []struct {
		array []uint32
		exp   int
	}{
		{
			array: []uint32{},
			exp:   0,
		},
		{
			array: []uint32{0},
			exp:   1,
		},
		{
			array: []uint32{1},
			exp:   1,
		},
		{
			array: []uint32{1, 2, 3, 5},
			exp:   2,
		},
		{
			array: []uint32{0, 1, 3, 9, 2048, 4096, 4097, 65534, 65535},
			exp:   6,
		},
		{
			array: []uint32{0, 10, 11, 12},
			exp:   2,
		},
	}

	for i, test := range tests {
		c.array = test.array
		ret := c.arrayCountRuns()
		if ret != test.exp {
			t.Fatalf("test #%v expected %v but got %v", i, test.exp, ret)
		}
	}
}
