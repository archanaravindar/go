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

func (x Int32x4) ToBits() (z Uint32x4) {
	for i := range z.vals {
		z.vals[i] = uint32(x.vals[i])
	}
	return
}
func (x Uint32x4) BitsToInt32() (z Int32x4) {
	for i := range z.vals {
		z.vals[i] = int32(x.vals[i])
	}
	return
}
func (x Int32x4) ConvertToUint32() (z Uint32x4) {
	for i := range z.vals {
		z.vals[i] = uint32(x.vals[i])
	}
	return
}
func (x Uint32x4) ConvertToInt32() (z Int32x4) {
	for i := range z.vals {
		z.vals[i] = int32(x.vals[i])
	}
	return
}

func (x Int64x2) Add(y Int64x2) (z Int64x2) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadInt64x2Array(p *[2]int64) (z Int64x2) { z.vals = *p; return }

func (x Int64x2) StoreArray(p *[2]int64) { *p = x.vals }

func (x Uint64x2) Add(y Uint64x2) (z Uint64x2) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadUint64x2Array(p *[2]uint64) (z Uint64x2) { z.vals = *p; return }

func (x Uint64x2) StoreArray(p *[2]uint64) { *p = x.vals }

func (x Int64x2) ToBits() (z Uint64x2) {
	for i := range z.vals {
		z.vals[i] = uint64(x.vals[i])
	}
	return
}
func (x Uint64x2) BitsToInt64() (z Int64x2) {
	for i := range z.vals {
		z.vals[i] = int64(x.vals[i])
	}
	return
}
func (x Int64x2) ConvertToUint64() (z Uint64x2) {
	for i := range z.vals {
		z.vals[i] = uint64(x.vals[i])
	}
	return
}
func (x Uint64x2) ConvertToInt64() (z Int64x2) {
	for i := range z.vals {
		z.vals[i] = int64(x.vals[i])
	}
	return
}

func (x Uint32x4) ReshapeToUint64s() (z Uint64x2) {
	z.vals[0] = uint64(x.vals[0]) | uint64(x.vals[1])<<32
	z.vals[1] = uint64(x.vals[2]) | uint64(x.vals[3])<<32
	return
}
func (x Uint64x2) ReshapeToUint32s() (z Uint32x4) {
	z.vals[0] = uint32(x.vals[0])
	z.vals[1] = uint32(x.vals[0] >> 32)
	z.vals[2] = uint32(x.vals[1])
	z.vals[3] = uint32(x.vals[1] >> 32)
	return
}

func (x Int16x8) Add(y Int16x8) (z Int16x8) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadInt16x8Array(p *[8]int16) (z Int16x8) { z.vals = *p; return }

func (x Int16x8) StoreArray(p *[8]int16) { *p = x.vals }

func (x Uint16x8) Add(y Uint16x8) (z Uint16x8) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadUint16x8Array(p *[8]uint16) (z Uint16x8) { z.vals = *p; return }

func (x Uint16x8) StoreArray(p *[8]uint16) { *p = x.vals }

func (x Uint16x8) ReshapeToUint32s() (z Uint32x4) {
	for i := range z.vals {
		z.vals[i] = uint32(x.vals[2*i]) |
			uint32(x.vals[2*i+1])<<16
	}
	return
}

func (x Uint32x4) ReshapeToUint16s() (z Uint16x8) {
	for i := range x.vals {
		z.vals[2*i] = uint16(x.vals[i])
		z.vals[2*i+1] = uint16(x.vals[i] >> 16)
	}
	return
}

func (x Uint16x8) ReshapeToUint64s() (z Uint64x2) {
	for i := range z.vals {
		z.vals[i] = uint64(x.vals[4*i]) |
			uint64(x.vals[4*i+1])<<16 |
			uint64(x.vals[4*i+2])<<32 |
			uint64(x.vals[4*i+3])<<48
	}
	return
}

func (x Uint64x2) ReshapeToUint16s() (z Uint16x8) {
	for i := range x.vals {
		z.vals[4*i] = uint16(x.vals[i])
		z.vals[4*i+1] = uint16(x.vals[i] >> 16)
		z.vals[4*i+2] = uint16(x.vals[i] >> 32)
		z.vals[4*i+3] = uint16(x.vals[i] >> 48)
	}
	return
}
func (x Int16x8) ToBits() (z Uint16x8) {
	for i := range z.vals {
		z.vals[i] = uint16(x.vals[i])
	}
	return
}

func (x Uint16x8) BitsToInt16() (z Int16x8) {
	for i := range z.vals {
		z.vals[i] = int16(x.vals[i])
	}
	return
}

func (x Int16x8) ConvertToUint16() (z Uint16x8) { return x.ToBits() }

func (x Uint16x8) ConvertToInt16() (z Int16x8) { return x.BitsToInt16() }

func (x Int8x16) Add(y Int8x16) (z Int8x16) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadInt8x16Array(p *[16]int8) (z Int8x16) { z.vals = *p; return }

func (x Int8x16) StoreArray(p *[16]int8) { *p = x.vals }

func (x Uint8x16) Add(y Uint8x16) (z Uint8x16) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadUint8x16Array(p *[16]uint8) (z Uint8x16) { z.vals = *p; return }

func (x Uint8x16) StoreArray(p *[16]uint8) { *p = x.vals }

func (x Int8x16) ToBits() (z Uint8x16) {
	for i := range z.vals {
		z.vals[i] = uint8(x.vals[i])
	}
	return
}

func (x Uint8x16) BitsToInt8() (z Int8x16) {
	for i := range z.vals {
		z.vals[i] = int8(x.vals[i])
	}
	return
}

func (x Int8x16) ConvertToUint8() (z Uint8x16) { return x.ToBits() }
func (x Uint8x16) ConvertToInt8() (z Int8x16) { return x.BitsToInt8() }

func (x Uint8x16) ReshapeToUint16s() (z Uint16x8) {
	for i := range z.vals {
		z.vals[i] = uint16(x.vals[2*i]) |
			uint16(x.vals[2*i+1])<<8
	}
	return
}

func (x Uint16x8) ReshapeToUint8s() (z Uint8x16) {
	for i := range x.vals {
		z.vals[2*i] = uint8(x.vals[i])
		z.vals[2*i+1] = uint8(x.vals[i] >> 8)
	}
	return
}

func (x Uint8x16) ReshapeToUint32s() (z Uint32x4) {
	for i := range z.vals {
		z.vals[i] = uint32(x.vals[4*i]) |
			uint32(x.vals[4*i+1])<<8 |
			uint32(x.vals[4*i+2])<<16 |
			uint32(x.vals[4*i+3])<<24
	}
	return
}

func (x Uint32x4) ReshapeToUint8s() (z Uint8x16) {
	for i := range x.vals {
		z.vals[4*i] = uint8(x.vals[i])
		z.vals[4*i+1] = uint8(x.vals[i] >> 8)
		z.vals[4*i+2] = uint8(x.vals[i] >> 16)
		z.vals[4*i+3] = uint8(x.vals[i] >> 24)
	}
	return
}

func (x Uint8x16) ReshapeToUint64s() (z Uint64x2) {
	for i := range z.vals {
		for j := range 8 {
			z.vals[i] |= uint64(x.vals[8*i+j]) << (8 * j)
		}
	}
	return
}

func (x Uint64x2) ReshapeToUint8s() (z Uint8x16) {
	for i := range x.vals {
		for j := range 8 {
			z.vals[8*i+j] = uint8(x.vals[i] >> (8 * j))
		}
	}
	return
}
