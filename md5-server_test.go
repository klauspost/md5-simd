// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/remeh/sizedwaitgroup"
	"hash"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type md5Test struct {
	in   string
	want string
}

var golden = []md5Test{
	{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "014842d480b571495a4a0363793f7367"},
	{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "0b649bcb5a82868817fec9a6e709d233"},
	{"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "bcd5708ed79b18f0f0aaa27fd0056d86"},
	{"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "e987c862fbd2f2f0ca859cb8d7806bf3"},
	{"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "982731671f0cd82cafce8d96a98e7a48"},
	{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "baf13e8b16d8c06324d7c9ab32cb7ff0"},
	{"gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg", "8ea3109cbd951bba1ace2f401a784ae4"},
	{"hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh", "d141045bfb385cad357e7c39c60e5da0"},
	{"", "d41d8cd98f00b204e9800998ecf8427e"},
	{"a", "0cc175b9c0f1b6a831c399e269772661"},
	{"ab", "187ef4436122d1cc2f40dc2b92f0eba0"},
	{"abc", "900150983cd24fb0d6963f7d28e17f72"},
	{"abcd", "e2fc714c4727ee9395f324cd2e7f331f"},
	{"abcde", "ab56b4d92b40713acc5af89985d4b786"},
	{"abcdef", "e80b5017098950fc58aad83c8c14978e"},
	{"abcdefg", "7ac66c0f148de9519b8bd264312c4d64"},
	{"abcdefgh", "e8dc4081b13434b45189a720b77b6818"},
	{"abcdefghi", "8aa99b1f439ff71293e95357bac6fd94"},
	{"abcdefghij", "a925576942e94b2ef57a066101b48876"},
	{"Discard medicine more than two years old.", "d747fc1719c7eacb84058196cfe56d57"},
	{"He who has a shady past knows that nice guys finish last.", "bff2dcb37ef3a44ba43ab144768ca837"},
	{"I wouldn't marry him with a ten foot pole.", "0441015ecb54a7342d017ed1bcfdbea5"},
	{"Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "9e3cac8e9e9757a60c3ea391130d3689"},
	{"The days of the digital watch are numbered.  -Tom Stoppard", "a0f04459b031f916a59a35cc482dc039"},
	{"Nepal premier won't resign.", "e7a48e0fe884faf31475d2a04b1362cc"},
	{"For every action there is an equal and opposite government program.", "637d2fe925c07c113800509964fb0e06"},
	{"His money is twice tainted: 'taint yours and 'taint mine.", "834a8d18d5c6562119cf4c7f5086cb71"},
	{"There is no reason for any individual to have a computer in their home. -Ken Olsen, 1977", "de3a4d2fd6c73ec2db2abad23b444281"},
	{"It's a tiny change to the code and not completely disgusting. - Bob Manchek", "acf203f997e2cf74ea3aff86985aefaf"},
	{"size:  a.out:  bad magic", "e1c1384cb4d2221dfdd7c795a4222c9a"},
	{"The major problem is with sendmail.  -Mark Horton", "c90f3ddecc54f34228c063d7525bf644"},
	{"Give me a rock, paper and scissors and I will move the world.  CCFestoon", "cdf7ab6c1fd49bd9933c43f3ea5af185"},
	{"If the enemy is within range, then so are you.", "83bc85234942fc883c063cbd7f0ad5d0"},
	{"It's well we cannot hear the screams/That we create in others' dreams.", "277cbe255686b48dd7e8f389394d9299"},
	{"You remind me of a TV show, but that's all right: I watch it anyway.", "fd3fb0a7ffb8af16603f3d3af98f8e1f"},
	{"C is as portable as Stonehedge!!", "469b13a78ebf297ecda64d4723655154"},
	{"Even if I could be Shakespeare, I think I should still choose to be Faraday. - A. Huxley", "63eb3a2f466410104731c4b037600110"},
	{"The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "72c2ed7592debca1c90fc0100f931a2f"},
	{"How can you write a big system without C++?  -Paul Glick", "132f7619d33b523b1d9e5bd8e0928355"},
	{"", "d41d8cd98f00b204e9800998ecf8427e"},
}

func testGolden16(t *testing.T, megabyte int) {

	server := NewMd5Server()
	h16 := [16]hash.Hash{}
	input := [16][]byte{}
	for i := range h16 {
		h16[i] = NewMd5(server)
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, megabyte*1024*1024)
	}

	for i := range h16 {
		h16[i].Write(input[i])
	}

	for i := range h16 {
		digest := h16[i].Sum([]byte{})
		got := fmt.Sprintf("%x\n", digest)

		h := md5.New()
		h.Write(input[i])
		want := fmt.Sprintf("%x\n", h.Sum(nil))

		if got != want {
			t.Errorf("TestGolden16[%d], got %v, want %v", i, got, want)
		}
	}
}

func TestGolden16(t *testing.T) {
	t.Run("1MB", func(t *testing.T) {
		testGolden16(t, 1)
	})
	t.Run("2MB", func(t *testing.T) {
		testGolden16(t, 2)
	})
}

func TestGolangGolden16(t *testing.T) {

	server := NewMd5Server()
	h16 := [16]hash.Hash{}
	for i := range h16 {
		h16[i] = NewMd5(server)
	}

	// Skip first 8, so we even 2 rounds of 16 test vectors
	golden16 := golden[8:]

	for tc := 0; tc < len(golden16); tc += 16 {
		for i := range h16 {
			h16[i].Reset()
			h16[i].Write([]byte(golden16[tc+i].in))
		}

		for i := range h16 {
			digest := h16[i].Sum([]byte{})
			if fmt.Sprintf("%x", digest) != golden16[tc+i].want {
				t.Errorf("TestGolangGolden[%d], got %v, want %v", tc+i, fmt.Sprintf("%x", digest), golden16[tc+i].want)
			}
		}
	}
}

func benchmarkGolden16(b *testing.B, blockSize int) {

	server := NewMd5Server()
	h16 := [16]hash.Hash{}
	input := [16][]byte{}
	for i := range h16 {
		h16[i] = NewMd5(server)
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, blockSize)
	}

	b.SetBytes(int64(blockSize * 16))
	b.ReportAllocs()
	b.ResetTimer()

	for j := 0; j < b.N; j++ {
		for i := range h16 {
			h16[i].Write(input[i])
		}
	}
}

func BenchmarkGolden16(b *testing.B) {

	if !hasAVX512 {
		b.SkipNow()
	}

	b.Run("32KB", func(b *testing.B) {
		benchmarkGolden16(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkGolden16(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkGolden16(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkGolden16(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkGolden16(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkGolden16(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkGolden16(b, 2*1024*1024)
	})
}

func benchmarkGoldenAvx2(b *testing.B, blockSize int) {

	server := NewMd5Server()
	h16 := [16]hash.Hash{}
	input := [16][]byte{}
	for i := range h16 {
		h16[i] = NewMd5(server)
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, blockSize)
	}

	const cores = 2 // AVX2 runs on two cores, so split effective performance in half
	b.SetBytes(int64(blockSize * 16 / cores))
	b.ReportAllocs()
	b.ResetTimer()

	for j := 0; j < b.N; j++ {
		for i := range h16 {
			h16[i].Write(input[i])
		}
	}
}

func BenchmarkGoldenAvx2(b *testing.B) {

	restore := hasAVX512

	// Make sure AVX512 is disabled
	hasAVX512 = false

	b.Run("32KB", func(b *testing.B) {
		benchmarkGoldenAvx2(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkGoldenAvx2(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkGoldenAvx2(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkGoldenAvx2(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkGoldenAvx2(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkGoldenAvx2(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkGoldenAvx2(b, 2*1024*1024)
	})

	hasAVX512 = restore
}

func benchmarkCryptoMd5(b *testing.B, blockSize int) {

	input := bytes.Repeat([]byte{0x61}, blockSize)

	b.SetBytes(int64(blockSize))
	b.ReportAllocs()
	b.ResetTimer()

	h := md5.New()

	for j := 0; j < b.N; j++ {
		h.Write(input)
	}
}

func BenchmarkCryptoMd5(b *testing.B) {
	b.Run("32KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 2*1024*1024)
	})
}

func testMd5Simulator(t *testing.T, concurrency, iterations, sizeVariation int, skipVerification bool, server *Md5Server) {

	rand.Seed(time.Now().UnixNano())
	verifier := make(map[string]string)

	mu := sync.Mutex{}

	swg := sizedwaitgroup.New(concurrency)
	for _i := 0; _i < iterations; _i++ {
		swg.Add()
		go func(i int) {
			defer swg.Done()
			h := NewMd5(server)
			mbs := 10 + rand.Intn(sizeVariation)
			h.Write(bytes.Repeat([]byte{0x61 + byte(i)}, mbs*1024*1024))
			digest := fmt.Sprintf("%x", h.Sum([]byte{}))
			mu.Lock()
			verifier[fmt.Sprintf("%d-%d", i, mbs)] = digest
			mu.Unlock()
		}(_i)
	}

	swg.Wait()

	if !skipVerification {
		fmt.Printf("Verifying %d entries...\n", len(verifier))

		swg = sizedwaitgroup.New(runtime.NumCPU())

		for _input, _digest := range verifier {

			swg.Add()
			go func(input, digest string) {
				defer swg.Done()

				p := strings.Split(input, "-")
				i, _ := strconv.Atoi(p[0])
				mbs, _ := strconv.Atoi(p[1])

				h := md5.New()
				h.Write(bytes.Repeat([]byte{0x61 + byte(i)}, mbs*1024*1024))
				d := fmt.Sprintf("%x", h.Sum([]byte{}))

				if digest != d {
					t.Errorf("testMd5Simulator[%s], got %s, want %s", input, digest, d)
				}
			}(_input, _digest)
		}
		swg.Wait()

		fmt.Println("Verification OK")
	}
}

func TestMd5Simulator(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	server := NewMd5Server()

	t.Run("", func(t *testing.T) {
		testMd5Simulator(t, 16, 1000, 100, false, server)
	})
}
