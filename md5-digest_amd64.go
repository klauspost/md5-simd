//+build !noasm,!appengine,gc

// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// md5Digest - Type for computing MD5 using either AVX2 or AVX512
type md5Digest struct {
	uid         uint64
	blocksCh    chan blockInput
	cycleServer chan uint64
	x           [BlockSize]byte
	nx          int
	len         uint64
}

// Size - Return size of checksum
func (d *md5Digest) Size() int { return Size }

// BlockSize - Return blocksize of checksum
func (d md5Digest) BlockSize() int { return BlockSize }

func (d *md5Digest) Reset() {
	if d.blocksCh == nil {
		panic("reset after close")
	}
	d.nx = 0
	d.len = 0
	d.sendBlock(blockInput{uid: d.uid, reset: true})
}

// write to digest
func (d *md5Digest) Write(p []byte) (nn int, err error) {
	if d.blocksCh == nil {
		return 0, errors.New("md5Digest closed")
	}

	// break input into chunks of maximum internalBlockSize size
	for {
		l := len(p)
		if l > internalBlockSize {
			l = internalBlockSize
		}
		nnn, err := d.write(p[:l])
		if err != nil {
			return nn, err
		}
		nn += nnn
		p = p[l:]

		if len(p) == 0 {
			break
		}

	}
	return
}

func (d *md5Digest) write(p []byte) (nn int, err error) {

	nn = len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == BlockSize {
			d.sendBlock(blockInput{uid: d.uid, msg: d.x[:]})
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= BlockSize {
		n := len(p) &^ (BlockSize - 1)
		d.sendBlock(blockInput{uid: d.uid, msg: p[:n]})
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

func (d *md5Digest) Close() {
	if d.blocksCh != nil {
		close(d.blocksCh)
		d.blocksCh = nil
	}
}

// Sum - Return MD5 sum in bytes
func (d *md5Digest) Sum(in []byte) (result []byte) {
	if d.blocksCh == nil {
		panic("sum after close")
	}

	trail := make([]byte, 0, 128)
	trail = append(trail, d.x[:d.nx]...)

	length := d.len
	// Padding.  Add a 1 bit and 0 bits until 56 bytes mod 64.
	var tmp [64]byte
	tmp[0] = 0x80
	if length%64 < 56 {
		trail = append(trail, tmp[0:56-length%64]...)
	} else {
		trail = append(trail, tmp[0:64+56-length%64]...)
	}

	// Length in bits.
	length <<= 3
	binary.LittleEndian.PutUint64(tmp[:], length) // append length in bits

	trail = append(trail, tmp[0:8]...)
	if len(trail)%BlockSize != 0 {
		panic(fmt.Errorf("internal error: sum block was not aligned. len=%d, nx=%d", len(trail), d.nx))
	}
	sumCh := make(chan sumResult, 1)
	d.sendBlock(blockInput{uid: d.uid, msg: trail, sumCh: sumCh})

	sum := <-sumCh

	return append(in, sum.digest[:]...)
}

func (d *md5Digest) sendBlock(bi blockInput) {
	select {
	case d.blocksCh <- bi:
		d.cycleServer <- d.uid
	}
}
