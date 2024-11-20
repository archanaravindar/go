// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bigmod

import (
	"errors"
	"internal/byteorder"
	"math/big"
	"math/bits"
)

const (
	// _W is the size in bits of our Limbs.
	_W = bits.UintSize
	// _S is the size in bytes of our Limbs.
	_S = _W / 8
)

// Choice represents a constant-time boolean. The value of Choice is always
// either 1 or 0. We use an int instead of bool in order to make decisions in
// constant time by turning it into a mask.
type Choice uint

func not(c Choice) Choice { return 1 ^ c }

const yes = Choice(1)
const no = Choice(0)

// ctMask is all 1s if on is yes, and all 0s otherwise.
func ctMask(on Choice) uint { return -uint(on) }

// ctEq returns 1 if x == y, and 0 otherwise. The execution time of this
// function does not depend on its inputs.
func ctEq(x, y uint) Choice {
	// If x != y, then either x - y or y - x will generate a carry.
	_, c1 := bits.Sub(x, y, 0)
	_, c2 := bits.Sub(y, x, 0)
	return not(Choice(c1 | c2))
}

// Nat represents an arbitrary natural number
//
// Each Nat has an announced length, which is the number of Limbs it has stored.
// Operations on this number are allowed to leak this length, but will not leak
// any information about the values contained in those Limbs.
type Nat struct {
	// Limbs is little-endian in base 2^W with W = bits.UintSize.
	Limbs []uint
}

// preallocTarget is the size in bits of the numbers used to implement the most
// common and most performant RSA key size. It's also enough to cover some of
// the operations of key sizes up to 4096.
const preallocTarget = 2048
const preallocLimbs = (preallocTarget + _W - 1) / _W

// NewNat returns a new nat with a size of zero, just like new(Nat), but with
// the preallocated capacity to hold a number of up to preallocTarget bits.
// NewNat inlines, so the allocation can live on the stack.
func NewNat() *Nat {
	Limbs := make([]uint, 0, preallocLimbs)
	return &Nat{Limbs}
}

// expand expands x to n Limbs, leaving its value unchanged.
func (x *Nat) expand(n int) *Nat {
	if len(x.Limbs) > n {
		panic("bigmod: internal error: shrinking nat")
	}
	if cap(x.Limbs) < n {
		newLimbs := make([]uint, n)
		copy(newLimbs, x.Limbs)
		x.Limbs = newLimbs
		return x
	}
	extraLimbs := x.Limbs[len(x.Limbs):n]
	clear(extraLimbs)
	x.Limbs = x.Limbs[:n]
	return x
}

func (x *Nat) clearWords(i, j uint) error{
	if i > j {
		return errors.New("invalid index in clearWords")
	}
	for k:=i; k<j; k++ {
		x.Limbs[k]=0
	}
	return nil
}
// reset returns a zero nat of n Limbs, reusing x's storage if n <= cap(x.Limbs).
func (x *Nat) Reset(n int) *Nat {
	if cap(x.Limbs) < n {
		x.Limbs = make([]uint, n)
		return x
	}
	clear(x.Limbs)
	x.Limbs = x.Limbs[:n]
	return x
}

// set assigns x = y, optionally resizing x to the appropriate size.
func (x *Nat) Set(y *Nat) *Nat {
	x.Reset(len(y.Limbs))
	copy(x.Limbs, y.Limbs)
	return x
}

// SetBig assigns x = n, optionally resizing n to the appropriate size.
//
// The announced length of x is set based on the actual bit size of the input,
// ignoring leading zeroes.
func (x *Nat) SetBig(n *big.Int) *Nat {
	Limbs := n.Bits()
	x.Reset(len(Limbs))
	for i := range Limbs {
		x.Limbs[i] = uint(Limbs[i])
	}
	return x
}

// Bytes returns x as a zero-extended big-endian byte slice. The size of the
// slice will match the size of m.
//
// x must have the same size as m and it must be reduced modulo m.
func (x *Nat) Bytes(m *Modulus) []byte {
	i := m.Size()
	bytes := make([]byte, i)
	for _, limb := range x.Limbs {
		for j := 0; j < _S; j++ {
			i--
			if i < 0 {
				if limb == 0 {
					break
				}
				panic("bigmod: modulus is smaller than nat")
			}
			bytes[i] = byte(limb)
			limb >>= 8
		}
	}
	return bytes
}

// resetToBytes assigns x = b, where b is a slice of big-endian bytes, resizing
// n to the appropriate size.
//
// The announced length of x is set based on the actual bit size of the input,
// ignoring leading zeroes.
func (x *Nat) resetToBytes(b []byte, m *Modulus) *Nat {
	x.Reset((len(b) + _S - 1) / _S)
	if err := x.setBytes(b,m); err != nil {
		panic("bigmod: internal error: bad arithmetic")
	}
	// Trim most significant (trailing in little-endian) zero Limbs.
	// We assume comparison with zero (but not the branch) is constant time.
	for i := len(x.Limbs) - 1; i >= 0; i-- {
		if x.Limbs[i] != 0 {
			break
		}
		x.Limbs = x.Limbs[:i]
	}
	return x
}
// SetBytes assigns x = b, where b is a slice of big-endian bytes.
// SetBytes returns an error if b >= m.
//
// The output will be resized to the size of m and overwritten.
func (x *Nat) SetBytes(b []byte, m *Modulus) (*Nat, error) {
	x.resetFor(m)
	if err := x.setBytes(b, m); err != nil {
		return nil, err
	}
	if x.cmpGeq(m.nat) == yes {
		return nil, errors.New("input overflows the modulus")
	}
	return x, nil
}

// SetOverflowingBytes assigns x = b, where b is a slice of big-endian bytes.
// SetOverflowingBytes returns an error if b has a longer bit length than m, but
// reduces overflowing values up to 2^⌈log2(m)⌉ - 1.
//
// The output will be resized to the size of m and overwritten.
func (x *Nat) SetOverflowingBytes(b []byte, m *Modulus) (*Nat, error) {
	x.resetFor(m)
	if err := x.setBytes(b, m); err != nil {
		return nil, err
	}
	leading := _W - bitLen(x.Limbs[len(x.Limbs)-1])
	if leading < m.leading {
		return nil, errors.New("input overflows the modulus size")
	}
	x.maybeSubtractModulus(no, m)
	return x, nil
}

// bigEndianUint returns the contents of buf interpreted as a
// big-endian encoded uint value.
func bigEndianUint(buf []byte) uint {
	if _W == 64 {
		return uint(byteorder.BeUint64(buf))
	}
	return uint(byteorder.BeUint32(buf))
}

func (x *Nat) setBytes(b []byte, m *Modulus) error {
	x.resetFor(m)
	i, k := len(b), 0
	for k < len(x.Limbs) && i >= _S {
		x.Limbs[k] = bigEndianUint(b[i-_S : i])
		i -= _S
		k++
	}
	for s := 0; s < _W && k < len(x.Limbs) && i > 0; s += 8 {
		x.Limbs[k] |= uint(b[i-1]) << s
		i--
	}
	if i > 0 {
		return errors.New("input overflows the modulus size")
	}
	return nil
}

// Equal returns 1 if x == y, and 0 otherwise.
//
// Both operands must have the same announced length.
func (x *Nat) Equal(y *Nat) Choice {
	// Eliminate bounds checks in the loop.
	size := len(x.Limbs)
	xLimbs := x.Limbs[:size]
	yLimbs := y.Limbs[:size]

	equal := yes
	for i := 0; i < size; i++ {
		equal &= ctEq(xLimbs[i], yLimbs[i])
	}
	return equal
}

// IsZero returns 1 if x == 0, and 0 otherwise.
func (x *Nat) IsZero() Choice {
	// Eliminate bounds checks in the loop.
	size := len(x.Limbs)
	xLimbs := x.Limbs[:size]

	zero := yes
	for i := 0; i < size; i++ {
		zero &= ctEq(xLimbs[i], 0)
	}
	return zero
}

// cmpGeq returns 1 if x >= y, and 0 otherwise.
//
// Both operands must have the same announced length.
func (x *Nat) cmpGeq(y *Nat) Choice {
	// Eliminate bounds checks in the loop.
	size := len(x.Limbs)
	xLimbs := x.Limbs[:size]
	yLimbs := y.Limbs[:size]

	var c uint
	for i := 0; i < size; i++ {
		_, c = bits.Sub(xLimbs[i], yLimbs[i], c)
	}
	// If there was a carry, then subtracting y underflowed, so
	// x is not greater than or equal to y.
	return not(Choice(c))
}

// assign sets x <- y if on == 1, and does nothing otherwise.
//
// Both operands must have the same announced length.
func (x *Nat) assign(on Choice, y *Nat) *Nat {
	// Eliminate bounds checks in the loop.
	size := len(x.Limbs)
	xLimbs := x.Limbs[:size]
	yLimbs := y.Limbs[:size]

	mask := ctMask(on)
	for i := 0; i < size; i++ {
		xLimbs[i] ^= mask & (xLimbs[i] ^ yLimbs[i])
	}
	return x
}

// add computes x += y and returns the carry.
//
// Both operands must have the same announced length.
func (x *Nat) add(y *Nat) (c uint) {
	// Eliminate bounds checks in the loop.
	size := len(x.Limbs)
	xLimbs := x.Limbs[:size]
	yLimbs := y.Limbs[:size]

	for i := 0; i < size; i++ {
		xLimbs[i], c = bits.Add(xLimbs[i], yLimbs[i], c)
	}
	return
}

// sub computes x -= y. It returns the borrow of the subtraction.
//
// Both operands must have the same announced length.
func (x *Nat) Sub(y *Nat) (c uint) {
	// Eliminate bounds checks in the loop.
	size := len(x.Limbs)
	xLimbs := x.Limbs[:size]
	yLimbs := y.Limbs[:size]

	for i := 0; i < size; i++ {
		xLimbs[i], c = bits.Sub(xLimbs[i], yLimbs[i], c)
	}
	return
}

// trailingZeroBits returns the number of consecutive least significant zero
// bits of x.
func (x *Nat) trailingZeroBits() uint {
        if x.Length() == 0 {
                return 0
        }
        var i uint
        for x.Limbs[i] == 0 {
                i++
        }
        // x[Limbs] != 0
        return i*_W + uint(bits.TrailingZeros(uint(x.Limbs[i])))
}


// Mul calculates z = x * y.
//
// All inputs should be the same length and already reduced modulo m.
// z will be resized to the size of m and overwritten.
func (z *Nat) Mul(x *Nat, y *Nat, m *Modulus) *Nat {
	n := len(m.nat.Limbs)
	zLimbs := z.resetFor(m).Limbs
	xLimbs := x.Limbs
	yLimbs := y.Limbs
	switch n {
	default:
		for i := 0; i < n; i++ {
			addMulVVW(zLimbs[i:], xLimbs, yLimbs[i])
		}
	case 2048 / _W:
		const n = 2048 / _W // compiler hint
		for i := 0; i < n; i++ {
			addMulVVW2048(&zLimbs[i:][0], &xLimbs[0], yLimbs[i])
		}
	}
	return z
}

// Modulus is used for modular arithmetic, precomputing relevant constants.
//
// Moduli are assumed to be odd numbers. Moduli can also leak the exact
// number of bits needed to store their value, and are stored without padding.
//
// Their actual value is still kept secret.
type Modulus struct {
	// The underlying natural number for this modulus.
	//
	// This will be stored without any padding, and shouldn't alias with any
	// other natural number being used.
	nat     *Nat
	leading int  // number of leading zeros in the modulus
	m0inv   uint // -nat.Limbs[0]⁻¹ mod _W
	rr      *Nat // R*R for montgomeryRepresentation
}

// rr returns R*R with R = 2^(_W * n) and n = len(m.nat.Limbs).
func rr(m *Modulus) *Nat {
	rr := NewNat().ExpandFor(m)
	n := uint(len(rr.Limbs))
	mLen := uint(m.BitLen())
	logR := _W * n

	// We start by computing R = 2^(_W * n) mod m. We can get pretty close, to
	// 2^⌊log₂m⌋, by setting the highest bit we can without having to reduce.
	rr.Limbs[n-1] = 1 << ((mLen - 1) % _W)
	// Then we double until we reach 2^(_W * n).
	for i := mLen - 1; i < logR; i++ {
		rr.Add(rr, m)
	}

	// Next we need to get from R to 2^(_W * n) R mod m (aka from one to R in
	// the Montgomery domain, meaning we can use Montgomery multiplication now).
	// We could do that by doubling _W * n times, or with a square-and-double
	// chain log2(_W * n) long. Turns out the fastest thing is to start out with
	// doublings, and switch to square-and-double once the exponent is large
	// enough to justify the cost of the multiplications.

	// The threshold is selected experimentally as a linear function of n.
	threshold := n / 4

	// We calculate how many of the most-significant bits of the exponent we can
	// compute before crossing the threshold, and we do it with doublings.
	i := bits.UintSize
	for logR>>i <= threshold {
		i--
	}
	for k := uint(0); k < logR>>i; k++ {
		rr.Add(rr, m)
	}

	// Then we process the remaining bits of the exponent with a
	// square-and-double chain.
	for i > 0 {
		rr.montgomeryMul(rr, rr, m)
		i--
		if logR>>i&1 != 0 {
			rr.Add(rr, m)
		}
	}

	return rr
}

// minusInverseModW computes -x⁻¹ mod _W with x odd.
//
// This operation is used to precompute a constant involved in Montgomery
// multiplication.
func minusInverseModW(x uint) uint {
	// Every iteration of this loop doubles the least-significant bits of
	// correct inverse in y. The first three bits are already correct (1⁻¹ = 1,
	// 3⁻¹ = 3, 5⁻¹ = 5, and 7⁻¹ = 7 mod 8), so doubling five times is enough
	// for 64 bits (and wastes only one iteration for 32 bits).
	//
	// See https://crypto.stackexchange.com/a/47496.
	y := x
	for i := 0; i < 5; i++ {
		y = y * (2 - x*y)
	}
	return -y
}

// NewModulusFromBig creates a new Modulus from a [big.Int].
//
// The Int must be odd. The number of significant bits (and nothing else) is
// leaked through timing side-channels.
func NewModulusFromBig(n *big.Int) (*Modulus, error) {
	if b := n.Bits(); len(b) == 0 {
		return nil, errors.New("modulus must be >= 0")
	} else if b[0]&1 != 1 {
		return nil, errors.New("modulus must be odd")
	}
	m := &Modulus{}
	m.nat = NewNat().SetBig(n)
	m.leading = _W - bitLen(m.nat.Limbs[len(m.nat.Limbs)-1])
	m.m0inv = minusInverseModW(m.nat.Limbs[0])
	m.rr = rr(m)
	return m, nil
}

func (x *Nat) Length() int {
	return len(x.Limbs)
}
func (z *Nat) GetWord(i uint) (uint, error) {
        if int(i) > len(z.Limbs) {
                return 0, errors.New("index greater than size")
        }
        return z.Limbs[i], nil
}

func (z *Nat) SetWord(i, w uint) (error) {
        if int(i) > len(z.Limbs) {
                return errors.New("index greater than size")
        }
        z.Limbs[i]=w
        return nil
}
func (x *Nat) slice(i, j uint) ([]uint, error) {
        if i<=j {
                return x.Limbs[i:j], nil
        } else {
                return nil, errors.New("invalid slice index")
        }
}       


func (x *Nat) maxLen(y *Nat) (uint, uint) {
	xl := len(x.Limbs)
	yl := len(y.Limbs)
	if xl > yl {
		return uint(xl), uint(xl*_W)
	} else {
		return uint(yl), uint(yl*_W)
	}
}
func(x *Nat) Or(i uint, w uint) error {
	if int(i)>len(x.Limbs) {
		return errors.New("index overflow error")
	}
	x.Limbs[i]|=w
	return nil
}

// bitLen is a version of bits.Len that only leaks the bit length of n, but not
// its value. bits.Len and bits.LeadingZeros use a lookup table for the
// low-order bits on some architectures.
func bitLen(n uint) int {
	var len int
	// We assume, here and elsewhere, that comparison to zero is constant time
	// with respect to different non-zero values.
	for n != 0 {
		len++
		n >>= 1
	}
	return len
}

// Size returns the size of m in bytes.
func (m *Modulus) Size() int {
	return (m.BitLen() + 7) / 8
}

// BitLen returns the size of m in bits.
func (m *Modulus) BitLen() int {
	return len(m.nat.Limbs)*_W - int(m.leading)
}

// Nat returns m as a Nat. The return value must not be written to.
func (m *Modulus) Nat() *Nat {
	return m.nat
}

// shiftIn calculates x = x << _W + y mod m.
//
// This assumes that x is already reduced mod m.
func (x *Nat) shiftIn(y uint, m *Modulus) *Nat {
	d := NewNat().resetFor(m)

	// Eliminate bounds checks in the loop.
	size := len(m.nat.Limbs)
	xLimbs := x.Limbs[:size]
	dLimbs := d.Limbs[:size]
	mLimbs := m.nat.Limbs[:size]

	// Each iteration of this loop computes x = 2x + b mod m, where b is a bit
	// from y. Effectively, it left-shifts x and adds y one bit at a time,
	// reducing it every time.
	//
	// To do the reduction, each iteration computes both 2x + b and 2x + b - m.
	// The next iteration (and finally the return line) will use either result
	// based on whether 2x + b overflows m.
	needSubtraction := no
	for i := _W - 1; i >= 0; i-- {
		carry := (y >> i) & 1
		var borrow uint
		mask := ctMask(needSubtraction)
		for i := 0; i < size; i++ {
			l := xLimbs[i] ^ (mask & (xLimbs[i] ^ dLimbs[i]))
			xLimbs[i], carry = bits.Add(l, l, carry)
			dLimbs[i], borrow = bits.Sub(xLimbs[i], mLimbs[i], borrow)
		}
		// Like in maybeSubtractModulus, we need the subtraction if either it
		// didn't underflow (meaning 2x + b > m) or if computing 2x + b
		// overflowed (meaning 2x + b > 2^_W*n > m).
		needSubtraction = not(Choice(borrow)) | Choice(carry)
	}
	return x.assign(needSubtraction, d)
}

// Mod calculates out = x mod m.
//
// This works regardless how large the value of x is.
//
// The output will be resized to the size of m and overwritten.
func (out *Nat) Mod(x *Nat, m *Modulus) *Nat {
	out.resetFor(m)
	// Working our way from the most significant to the least significant limb,
	// we can insert each limb at the least significant position, shifting all
	// previous Limbs left by _W. This way each limb will get shifted by the
	// correct number of bits. We can insert at least N - 1 Limbs without
	// overflowing m. After that, we need to reduce every time we shift.
	i := len(x.Limbs) - 1
	// For the first N - 1 Limbs we can skip the actual shifting and position
	// them at the shifted position, which starts at min(N - 2, i).
	start := len(m.nat.Limbs) - 2
	if i < start {
		start = i
	}
	for j := start; j >= 0; j-- {
		out.Limbs[j] = x.Limbs[i]
		i--
	}
	// We shift in the remaining Limbs, reducing modulo m each time.
	for i >= 0 {
		out.shiftIn(x.Limbs[i], m)
		i--
	}
	return out
}

// ExpandFor ensures x has the right size to work with operations modulo m.
//
// The announced size of x must be smaller than or equal to that of m.
func (x *Nat) ExpandFor(m *Modulus) *Nat {
	return x.expand(len(m.nat.Limbs))
}

// resetFor ensures out has the right size to work with operations modulo m.
//
// out is zeroed and may start at any size.
func (out *Nat) resetFor(m *Modulus) *Nat {
	return out.Reset(len(m.nat.Limbs))
}

// maybeSubtractModulus computes x -= m if and only if x >= m or if "always" is yes.
//
// It can be used to reduce modulo m a value up to 2m - 1, which is a common
// range for results computed by higher level operations.
//
// always is usually a carry that indicates that the operation that produced x
// overflowed its size, meaning abstractly x > 2^_W*n > m even if x < m.
//
// x and m operands must have the same announced length.
func (x *Nat) maybeSubtractModulus(always Choice, m *Modulus) {
	t := NewNat().Set(x)
	underflow := t.Sub(m.nat)
	// We keep the result if x - m didn't underflow (meaning x >= m)
	// or if always was set.
	keep := not(Choice(underflow)) | Choice(always)
	x.assign(keep, t)
}

// Sub computes x = x - y mod m.
//
// The length of both operands must be the same as the modulus. Both operands
// must already be reduced modulo m.
func (x *Nat) SubMod(y *Nat, m *Modulus) *Nat {
	underflow := x.Sub(y)
	// If the subtraction underflowed, add m.
	t := NewNat().Set(x)
	t.add(m.nat)
	x.assign(Choice(underflow), t)
	return x
}

// Add computes x = x + y mod m.
//
// The length of both operands must be the same as the modulus. Both operands
// must already be reduced modulo m.
func (x *Nat) Add(y *Nat, m *Modulus) *Nat {
	overflow := x.add(y)
	x.maybeSubtractModulus(Choice(overflow), m)
	return x
}

// montgomeryRepresentation calculates x = x * R mod m, with R = 2^(_W * n) and
// n = len(m.nat.Limbs).
//
// Faster Montgomery multiplication replaces standard modular multiplication for
// numbers in this representation.
//
// This assumes that x is already reduced mod m.
func (x *Nat) montgomeryRepresentation(m *Modulus) *Nat {
	// A Montgomery multiplication (which computes a * b / R) by R * R works out
	// to a multiplication by R, which takes the value out of the Montgomery domain.
	return x.montgomeryMul(x, m.rr, m)
}

// montgomeryReduction calculates x = x / R mod m, with R = 2^(_W * n) and
// n = len(m.nat.Limbs).
//
// This assumes that x is already reduced mod m.
func (x *Nat) montgomeryReduction(m *Modulus) *Nat {
	// By Montgomery multiplying with 1 not in Montgomery representation, we
	// convert out back from Montgomery representation, because it works out to
	// dividing by R.
	one := NewNat().ExpandFor(m)
	one.Limbs[0] = 1
	return x.montgomeryMul(x, one, m)
}

// montgomeryMul calculates x = a * b / R mod m, with R = 2^(_W * n) and
// n = len(m.nat.Limbs), also known as a Montgomery multiplication.
//
// All inputs should be the same length and already reduced modulo m.
// x will be resized to the size of m and overwritten.
func (x *Nat) montgomeryMul(a *Nat, b *Nat, m *Modulus) *Nat {
	n := len(m.nat.Limbs)
	mLimbs := m.nat.Limbs[:n]
	aLimbs := a.Limbs[:n]
	bLimbs := b.Limbs[:n]

	switch n {
	default:
		// Attempt to use a stack-allocated backing array.
		T := make([]uint, 0, preallocLimbs*2)
		if cap(T) < n*2 {
			T = make([]uint, 0, n*2)
		}
		T = T[:n*2]

		// This loop implements Word-by-Word Montgomery Multiplication, as
		// described in Algorithm 4 (Fig. 3) of "Efficient Software
		// Implementations of Modular Exponentiation" by Shay Gueron
		// [https://eprint.iacr.org/2011/239.pdf].
		var c uint
		for i := 0; i < n; i++ {
			_ = T[n+i] // bounds check elimination hint

			// Step 1 (T = a × b) is computed as a large pen-and-paper column
			// multiplication of two numbers with n base-2^_W digits. If we just
			// wanted to produce 2n-wide T, we would do
			//
			//   for i := 0; i < n; i++ {
			//       d := bLimbs[i]
			//       T[n+i] = addMulVVW(T[i:n+i], aLimbs, d)
			//   }
			//
			// where d is a digit of the multiplier, T[i:n+i] is the shifted
			// position of the product of that digit, and T[n+i] is the final carry.
			// Note that T[i] isn't modified after processing the i-th digit.
			//
			// Instead of running two loops, one for Step 1 and one for Steps 2–6,
			// the result of Step 1 is computed during the next loop. This is
			// possible because each iteration only uses T[i] in Step 2 and then
			// discards it in Step 6.
			d := bLimbs[i]
			c1 := addMulVVW(T[i:n+i], aLimbs, d)

			// Step 6 is replaced by shifting the virtual window we operate
			// over: T of the algorithm is T[i:] for us. That means that T1 in
			// Step 2 (T mod 2^_W) is simply T[i]. k0 in Step 3 is our m0inv.
			Y := T[i] * m.m0inv

			// Step 4 and 5 add Y × m to T, which as mentioned above is stored
			// at T[i:]. The two carries (from a × d and Y × m) are added up in
			// the next word T[n+i], and the carry bit from that addition is
			// brought forward to the next iteration.
			c2 := addMulVVW(T[i:n+i], mLimbs, Y)
			T[n+i], c = bits.Add(c1, c2, c)
		}

		// Finally for Step 7 we copy the final T window into x, and subtract m
		// if necessary (which as explained in maybeSubtractModulus can be the
		// case both if x >= m, or if x overflowed).
		//
		// The paper suggests in Section 4 that we can do an "Almost Montgomery
		// Multiplication" by subtracting only in the overflow case, but the
		// cost is very similar since the constant time subtraction tells us if
		// x >= m as a side effect, and taking care of the broken invariant is
		// highly undesirable (see https://go.dev/issue/13907).
		copy(x.Reset(n).Limbs, T[n:])
		x.maybeSubtractModulus(Choice(c), m)

	// The following specialized cases follow the exact same algorithm, but
	// optimized for the sizes most used in RSA. addMulVVW is implemented in
	// assembly with loop unrolling depending on the architecture and bounds
	// checks are removed by the compiler thanks to the constant size.
	case 1024 / _W:
		const n = 1024 / _W // compiler hint
		T := make([]uint, n*2)
		var c uint
		for i := 0; i < n; i++ {
			d := bLimbs[i]
			c1 := addMulVVW1024(&T[i], &aLimbs[0], d)
			Y := T[i] * m.m0inv
			c2 := addMulVVW1024(&T[i], &mLimbs[0], Y)
			T[n+i], c = bits.Add(c1, c2, c)
		}
		copy(x.Reset(n).Limbs, T[n:])
		x.maybeSubtractModulus(Choice(c), m)

	case 1536 / _W:
		const n = 1536 / _W // compiler hint
		T := make([]uint, n*2)
		var c uint
		for i := 0; i < n; i++ {
			d := bLimbs[i]
			c1 := addMulVVW1536(&T[i], &aLimbs[0], d)
			Y := T[i] * m.m0inv
			c2 := addMulVVW1536(&T[i], &mLimbs[0], Y)
			T[n+i], c = bits.Add(c1, c2, c)
		}
		copy(x.Reset(n).Limbs, T[n:])
		x.maybeSubtractModulus(Choice(c), m)

	case 2048 / _W:
		const n = 2048 / _W // compiler hint
		T := make([]uint, n*2)
		var c uint
		for i := 0; i < n; i++ {
			d := bLimbs[i]
			c1 := addMulVVW2048(&T[i], &aLimbs[0], d)
			Y := T[i] * m.m0inv
			c2 := addMulVVW2048(&T[i], &mLimbs[0], Y)
			T[n+i], c = bits.Add(c1, c2, c)
		}
		copy(x.Reset(n).Limbs, T[n:])
		x.maybeSubtractModulus(Choice(c), m)
	}

	return x
}

// addMulVVW multiplies the multi-word value x by the single-word value y,
// adding the result to the multi-word value z and returning the final carry.
// It can be thought of as one row of a pen-and-paper column multiplication.
func addMulVVW(z, x []uint, y uint) (carry uint) {
	_ = x[len(z)-1] // bounds check elimination hint
	for i := range z {
		hi, lo := bits.Mul(x[i], y)
		lo, c := bits.Add(lo, z[i], 0)
		// We use bits.Add with zero to get an add-with-carry instruction that
		// absorbs the carry from the previous bits.Add.
		hi, _ = bits.Add(hi, 0, c)
		lo, c = bits.Add(lo, carry, 0)
		hi, _ = bits.Add(hi, 0, c)
		carry = hi
		z[i] = lo
	}
	return carry
}

// Mul calculates x = x * y mod m.
//
// The length of both operands must be the same as the modulus. Both operands
// must already be reduced modulo m.
func (x *Nat) MulMod(y *Nat, m *Modulus) *Nat {
	// A Montgomery multiplication by a value out of the Montgomery domain
	// takes the result out of Montgomery representation.
	xR := NewNat().Set(x).montgomeryRepresentation(m) // xR = x * R mod m
	return x.montgomeryMul(xR, y, m)                  // x = xR * y / R mod m
}

// Exp calculates out = x^e mod m.
//
// The exponent e is represented in big-endian order. The output will be resized
// to the size of m and overwritten. x must already be reduced modulo m.
func (out *Nat) Exp(x *Nat, e []byte, m *Modulus) *Nat {
	// We use a 4 bit window. For our RSA workload, 4 bit windows are faster
	// than 2 bit windows, but use an extra 12 nats worth of scratch space.
	// Using bit sizes that don't divide 8 are more complex to implement, but
	// are likely to be more efficient if necessary.

	table := [(1 << 4) - 1]*Nat{ // table[i] = x ^ (i+1)
		// newNat calls are unrolled so they are allocated on the stack.
		NewNat(), NewNat(), NewNat(), NewNat(), NewNat(),
		NewNat(), NewNat(), NewNat(), NewNat(), NewNat(),
		NewNat(), NewNat(), NewNat(), NewNat(), NewNat(),
	}
	table[0].Set(x).montgomeryRepresentation(m)
	for i := 1; i < len(table); i++ {
		table[i].montgomeryMul(table[i-1], table[0], m)
	}

	out.resetFor(m)
	out.Limbs[0] = 1
	out.montgomeryRepresentation(m)
	tmp := NewNat().ExpandFor(m)
	for _, b := range e {
		for _, j := range []int{4, 0} {
			// Square four times. Optimization note: this can be implemented
			// more efficiently than with generic Montgomery multiplication.
			out.montgomeryMul(out, out, m)
			out.montgomeryMul(out, out, m)
			out.montgomeryMul(out, out, m)
			out.montgomeryMul(out, out, m)

			// Select x^k in constant time from the table.
			k := uint((b >> j) & 0b1111)
			for i := range table {
				tmp.assign(ctEq(k, uint(i+1)), table[i])
			}

			// Multiply by x^k, discarding the result if k = 0.
			tmp.montgomeryMul(out, tmp, m)
			out.assign(not(ctEq(k, 0)), tmp)
		}
	}

	return out.montgomeryReduction(m)
}

// NewModulus creates a new Modulus from a slice of big-endian bytes.
//
// The value must be odd. The number of significant bits (and nothing else) is
// leaked through timing side-channels.
func NewModulus(b []byte) (*Modulus, error) {
	if len(b) == 0 || b[len(b)-1]&1 != 1 {
		return nil, errors.New("modulus must be > 0 and odd")
	}
	m := &Modulus{}
	m.nat = NewNat().resetToBytes(b, m)
	m.leading = _W - bitLen(m.nat.Limbs[len(m.nat.Limbs)-1])
	m.m0inv = minusInverseModW(m.nat.Limbs[0])
	m.rr = rr(m)
	return m, nil
}
// ExpShortVarTime calculates out = x^e mod m.
//
// The output will be resized to the size of m and overwritten. x must already
// be reduced modulo m. This leaks the exponent through timing side-channels.
func (out *Nat) ExpShortVarTime(x *Nat, e uint, m *Modulus) *Nat {
	// For short exponents, precomputing a table and using a window like in Exp
	// doesn't pay off. Instead, we do a simple conditional square-and-multiply
	// chain, skipping the initial run of zeroes.
	xR := NewNat().Set(x).montgomeryRepresentation(m)
	out.Set(xR)
	for i := bits.UintSize - bitLen(e) + 1; i < bits.UintSize; i++ {
		out.montgomeryMul(out, out, m)
		if k := (e >> (bits.UintSize - i - 1)) & 1; k != 0 {
			out.montgomeryMul(out, xR, m)
		}
	}
	return out.montgomeryReduction(m)
}
