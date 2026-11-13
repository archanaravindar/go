// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build goexperiment.simd && ppc64le && !ppc64le.power9

package archsimd

func (x Int32x4) Add(y Int32x4) (z Int32x4) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadInt32x4Array(p *[4]int32) (z Int32x4) { z.vals = *p; return }

func (x Int32x4) StoreArray(p *[4]int32) { *p = x.vals }

func (x Uint32x4) Add(y Uint32x4) (z Uint32x4) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadUint32x4Array(p *[4]uint32) (z Uint32x4) { z.vals = *p; return }

func (x Uint32x4) StoreArray(p *[4]uint32) { *p = x.vals }

func (x Int32x4) ToBits() (z Uint32x4)          { for i := range z.vals { z.vals[i] = uint32(x.vals[i]) }; return }
func (x Uint32x4) BitsToInt32() (z Int32x4)     { for i := range z.vals { z.vals[i] = int32(x.vals[i]) }; return }
func (x Int32x4) ConvertToUint32() (z Uint32x4) { for i := range z.vals { z.vals[i] = uint32(x.vals[i]) }; return }
func (x Uint32x4) ConvertToInt32() (z Int32x4)  { for i := range z.vals { z.vals[i] = int32(x.vals[i]) }; return }


