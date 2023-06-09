// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builder_test

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	. "github.com/weiwenchen2022/builder"
)

func check(t *testing.T, b *Builder, want string) {
	t.Helper()

	got := b.String()
	if want != got {
		t.Errorf("String: got %#q; want %#q", got, want)
	}

	if n := b.Len(); len(got) != n {
		t.Errorf("Len: got %d; but len(String()) is %d", n, len(got))
	}
	if n := b.Cap(); n < len(got) {
		t.Errorf("Cap: got %d; but len(String()) is %d", n, len(got))
	}
}

func TestBuilder(t *testing.T) {
	t.Parallel()

	var b Builder
	check(t, &b, "")

	n, err := b.WriteString("hello")
	if err != nil || n != 5 {
		t.Errorf("WriteString: got %d,%s; want 5,nil", n, err)
	}
	check(t, &b, "hello")

	if err := b.WriteByte(' '); err != nil {
		t.Errorf("WriteByte: %s", err)
	}
	check(t, &b, "hello ")

	n, err = b.WriteString("world")
	if err != nil || n != 5 {
		t.Errorf("WriteString: got %d,%s; want 5,nil", n, err)
	}
	check(t, &b, "hello world")
}

func TestBuilderString(t *testing.T) {
	t.Parallel()

	var b Builder
	b.WriteString("alpha")
	check(t, &b, "alpha")
	s1 := b.String()
	b.WriteString("beta")
	check(t, &b, "alphabeta")
	s2 := b.String()
	b.WriteString("gamma")
	check(t, &b, "alphabetagamma")
	s3 := b.String()

	// Check that subsequent operations didn't change the returned strings.
	if want := "alpha"; want != s1 {
		t.Errorf("first String result is now %q; want %q", s1, want)
	}
	if want := "alphabeta"; want != s2 {
		t.Errorf("second String result is now %q; want %q", s2, want)
	}
	if want := "alphabetagamma"; want != s3 {
		t.Errorf("third String result is now %q; want %q", s3, want)
	}
}

func TestBuilderReset(t *testing.T) {
	t.Parallel()

	var b Builder
	check(t, &b, "")

	b.WriteString("aaa")
	s := b.String()
	check(t, &b, "aaa")
	b.Reset()
	check(t, &b, "")

	// Ensure that writing after Reset doesn't alter
	// previously returned strings.
	b.WriteString("bbb")
	check(t, &b, "bbb")
	if want := "aaa"; want != s {
		t.Errorf("previous String result changed after Reset: got %q; want %q", s, want)
	}
}

func TestBuilderGrow(t *testing.T) {
	for _, growLen := range []int{0, 100, 1000, 10000, 100000} {
		s := strings.Repeat("a", growLen)
		allocs := testing.AllocsPerRun(100, func() {
			var b strings.Builder
			b.Grow(growLen) // should be only alloc, when growLen > 0
			if b.Cap() < growLen {
				t.Fatalf("growLen=%d: Cap() is lower than growLen", growLen)
			}

			b.WriteString(s)
			if s != b.String() {
				t.Fatalf("growLen=%d: bad data written after Grow", growLen)
			}
		})

		wantAllocs := 1
		if growLen == 0 {
			wantAllocs = 0
		}
		if g := int(allocs); wantAllocs != g {
			t.Errorf("growLen=%d: got %d allocs during Write; want %d", growLen, g, wantAllocs)
		}
	}

	// when growLen < 0, should panic
	var b Builder
	defer func() {
		if r := recover(); r == nil {
			t.Error("b.Grow(-1) should panic()")
		}
	}()
	b.Grow(-1)
}

func TestBuilderClip(t *testing.T) {
	t.Parallel()

	var b Builder
	b.Grow(6)
	b.WriteString(strings.Repeat("a", 3))

	if n := b.Len(); n != 3 {
		t.Errorf("Len: got %d, want 3", n)
	}
	if n := b.Cap(); n < 6 {
		t.Errorf("Cap: got %d, want >= 6", n)
	}

	s1 := b.String()
	b.Clip()
	if s2 := b.String(); s1 != s2 {
		t.Errorf("%q.Clip() = %q, want %[1]q", s1, s2)
	}
	if n := b.Cap(); n != 3 {
		t.Errorf("Cap: got %d, want 3", n)
	}
}

func TestBuilderWrite2(t *testing.T) {
	t.Parallel()

	const s = "hello 世界"
	tests := []struct {
		name string
		fn   func(b *Builder) (int, error)
		n    int
		want string
	}{
		{
			"Write",
			func(b *Builder) (int, error) { return b.Write([]byte(s)) },
			len(s),
			s,
		},
		{
			"WriteRune",
			func(b *Builder) (int, error) { return b.WriteRune('a') },
			1,
			"a",
		},
		{
			"WriteRuneWide",
			func(b *Builder) (int, error) { return b.WriteRune('世') },
			3,
			"世",
		},
		{
			"WriteString",
			func(b *Builder) (int, error) { return b.WriteString(s) },
			len(s),
			s,
		},
		{
			"WriteBool",
			func(b *Builder) (int, error) { return b.WriteBool(true) },
			4,
			"true",
		},
		{
			"WriteIntBase10",
			func(b *Builder) (int, error) { return b.WriteInt(-42, 10) },
			3,
			"-42",
		},
		{
			"WriteIntBase16",
			func(b *Builder) (int, error) { return b.WriteInt(-42, 16) },
			3,
			"-2a",
		},
		{
			"WriteUintBase10",
			func(b *Builder) (int, error) { return b.WriteUint(42, 10) },
			2,
			"42",
		},
		{
			"WriteUintBase16",
			func(b *Builder) (int, error) { return b.WriteUint(42, 16) },
			2,
			"2a",
		},
		{
			"WriteFloat32",
			func(b *Builder) (int, error) { return b.WriteFloat(3.1415926535, 'E', -1, 32) },
			len("3.1415927E+00"),
			"3.1415927E+00",
		},
		{
			"WriteFloat64",
			func(b *Builder) (int, error) { return b.WriteFloat(3.1415926535, 'E', -1, 64) },
			len("3.1415926535E+00"),
			"3.1415926535E+00",
		},
		{
			"WriteQuote",
			func(b *Builder) (int, error) { return b.WriteQuote(`"Fran & Freddie's Diner"`) },
			len(strconv.Quote("\"Fran & Freddie's Diner\"")),
			strconv.Quote("\"Fran & Freddie's Diner\""),
		},
		{
			"WriteQuoteRune",
			func(b *Builder) (int, error) { return b.WriteQuoteRune('☺') },
			len(strconv.QuoteRune('☺')),
			strconv.QuoteRune('☺'),
		},
		{
			"WriteQuoteRuneToASCII",
			func(b *Builder) (int, error) { return b.WriteQuoteRuneToASCII('☺') },
			len(strconv.QuoteRuneToASCII('☺')),
			strconv.QuoteRuneToASCII('☺'),
		},
		{
			"WriteQuoteRuneToGraphic",
			func(b *Builder) (int, error) { return b.WriteQuoteRuneToGraphic('☺') },
			len(strconv.QuoteRuneToGraphic('☺')),
			strconv.QuoteRuneToGraphic('☺'),
		},
		{
			"WriteQuoteRuneToGraphic2",
			func(b *Builder) (int, error) { return b.WriteQuoteRuneToGraphic('\u263a') },
			len(strconv.QuoteRuneToGraphic('\u263a')),
			strconv.QuoteRuneToGraphic('\u263a'),
		},
		{
			"WriteQuoteRuneToGraphic3",
			func(b *Builder) (int, error) { return b.WriteQuoteRuneToGraphic('\u000a') },
			len(strconv.QuoteRuneToGraphic('\u000a')),
			strconv.QuoteRuneToGraphic('\u000a'),
		},
		{
			"WriteQuoteRuneToGraphic4",
			func(b *Builder) (int, error) { return b.WriteQuoteRuneToGraphic('	') }, // tab character
			len(strconv.QuoteRuneToGraphic('	')),
			strconv.QuoteRuneToGraphic('	'),
		},
		{
			"WriteQuoteToASCII",
			func(b *Builder) (int, error) { return b.WriteQuoteToASCII(`"Fran & Freddie's Diner"`) },
			len(strconv.QuoteToASCII("\"Fran & Freddie's Diner\"")),
			strconv.QuoteToASCII("\"Fran & Freddie's Diner\""),
		},
		{
			"WriteQuoteToGraphic",
			func(b *Builder) (int, error) { return b.WriteQuoteToGraphic("This is a \u263a	\u000a") },
			len(strconv.QuoteToGraphic("This is a \u263a	\u000a")),
			strconv.QuoteToGraphic("This is a \u263a	\u000a"),
		},
		{
			"WriteQuoteToGraphic2",
			func(b *Builder) (int, error) { return b.WriteQuoteToGraphic(`" This is a ☺ \n "`) },
			len(strconv.QuoteToGraphic(`" This is a ☺ \n "`)),
			strconv.QuoteToGraphic(`" This is a ☺ \n "`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b Builder
			n, err := tt.fn(&b)
			if err != nil {
				t.Fatalf("first call: got %v", err)
			}
			if tt.n != n {
				t.Errorf("first call: got n=%d; want %d", n, tt.n)
			}
			check(t, &b, tt.want)

			n, err = tt.fn(&b)
			if err != nil {
				t.Fatalf("second call: got %v", err)
			}
			if tt.n != n {
				t.Errorf("second call: got n=%d; want %d", n, tt.n)
			}
			check(t, &b, tt.want+tt.want)
		})
	}
}

func TestBuilderWriteByte(t *testing.T) {
	t.Parallel()

	var b Builder
	if err := b.WriteByte('a'); err != nil {
		t.Error(err)
	}
	if err := b.WriteByte(0); err != nil {
		t.Error(err)
	}
	check(t, &b, "a\x00")
}

func TestBuilderAllocs(t *testing.T) {
	// Issue 23382; verify that copyCheck doesn't force the
	// Builder to escape and be heap allocated.
	n := testing.AllocsPerRun(10000, func() {
		var b Builder
		b.Grow(5)
		b.WriteString("abcde")
		_ = b.String()
	})
	if n != 1 {
		t.Errorf("Builder allocs = %v; want 1", n)
	}
}

func TestBuilderCopyPanic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fn        func()
		wantPanic bool
	}{
		{
			name: "String",
			fn: func() {
				var a Builder
				_ = a.WriteByte('x')
				b := a
				_ = b.String() // appease vet
			},
			wantPanic: false,
		},
		{
			name: "Len",
			fn: func() {
				var a Builder
				_ = a.WriteByte('x')
				b := a
				b.Len()
			},
			wantPanic: false,
		},
		{
			name: "Cap",
			fn: func() {
				var a Builder
				_ = a.WriteByte('x')
				b := a
				b.Cap()
			},
			wantPanic: false,
		},
		{
			name: "Reset",
			fn: func() {
				var a Builder
				_ = a.WriteByte('x')
				b := a
				b.Reset()
				_ = b.WriteByte('y')
			},
			wantPanic: false,
		},
		{
			name: "Write",
			fn: func() {
				var a Builder
				_, _ = a.Write([]byte("x"))
				b := a
				_, _ = b.Write([]byte("y"))
			},
			wantPanic: true,
		},
		{
			name: "WriteByte",
			fn: func() {
				var a Builder
				_ = a.WriteByte('x')
				b := a
				_ = b.WriteByte('y')
			},
			wantPanic: true,
		},
		{
			name: "WriteString",
			fn: func() {
				var a Builder
				_, _ = a.WriteString("x")
				b := a
				_, _ = b.WriteString("y")
			},
			wantPanic: true,
		},
		{
			name: "WriteRune",
			fn: func() {
				var a Builder
				_, _ = a.WriteRune('x')
				b := a
				_, _ = b.WriteRune('y')
			},
			wantPanic: true,
		},
		{
			name: "Grow",
			fn: func() {
				var a Builder
				a.Grow(1)
				b := a
				b.Grow(2)
			},
			wantPanic: true,
		},
		{
			name: "Clip",
			fn: func() {
				var a Builder
				a.Grow(1)
				b := a
				b.Clip()
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		didPanic := make(chan bool, 1)
		go func() {
			fnPanic := true // If fn panics fnPanic will remain true.
			defer func() { _ = recover(); didPanic <- fnPanic }()
			tt.fn()
			fnPanic = false
		}()
		if got := <-didPanic; tt.wantPanic != got {
			t.Errorf("%s: panicked = %t; want %t", tt.name, got, tt.wantPanic)
		}
	}
}

func TestBuilderWriteInvalidRune(t *testing.T) {
	// Invalid runes, including negative ones, should be written as
	// utf8.RuneError.
	for _, r := range []rune{-1, utf8.MaxRune + 1} {
		var b Builder
		b.WriteRune(r)
		check(t, &b, "\uFFFD")
	}
}

var someBytes = []byte("some bytes sdljlk jsklj3lkjlk djlkjw")

type builderInterface interface {
	Grow(n int)
	Write(p []byte) (int, error)
	String() string
}

func benchmarkBuilder(b *testing.B, f func(b *testing.B, buf builderInterface, numWrite int, grow bool)) {
	b.Run("1Write_NoGrow", func(b *testing.B) {
		for _, buf := range []builderInterface{&strings.Builder{}, &Builder{}} {
			b.Run(fmt.Sprintf("%T", buf), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					buf = reflect.New(reflect.TypeOf(buf).Elem()).Interface().(builderInterface)
					f(b, buf, 1, false)
				}
			})
		}
	})
	b.Run("3Write_NoGrow", func(b *testing.B) {
		for _, buf := range []builderInterface{&strings.Builder{}, &Builder{}} {
			b.Run(fmt.Sprintf("%T", buf), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					buf = reflect.New(reflect.TypeOf(buf).Elem()).Interface().(builderInterface)
					f(b, buf, 3, false)
				}
			})
		}
	})
	b.Run("3Write_Grow", func(b *testing.B) {
		for _, buf := range []builderInterface{&strings.Builder{}, &Builder{}} {
			b.Run(fmt.Sprintf("%T", buf), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					buf = reflect.New(reflect.TypeOf(buf).Elem()).Interface().(builderInterface)
					f(b, buf, 3, true)
				}
			})
		}
	})
}

var sinkS string

func BenchmarkBuildString_Builder(b *testing.B) {
	benchmarkBuilder(b, func(b *testing.B, buf builderInterface, numWrite int, grow bool) {
		if grow {
			buf.Grow(len(someBytes) * numWrite)
		}
		for j := 0; j < numWrite; j++ {
			buf.Write(someBytes)
		}
		sinkS = buf.String()
	})
}
