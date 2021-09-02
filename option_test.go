package set

import (
	"testing"
)

func Test_entryGrowWork(t *testing.T) {
	s := NewOption31(100)
	for i := 0; i < 120; i++ {
		s.Store(uint32(i))
	}

	type args struct {
		s   *Option
		old *entry
		cap uint32
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				s:   s,
				old: s.getEntry(),
				cap: 200,
			},
			want: true,
		},
		{
			name: "",
			args: args{
				s:   s,
				old: s.getEntry(),
				cap: 20,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldLen := Size(s)
			if got := optGrowWork(tt.args.s, tt.args.old, tt.args.cap); got != tt.want {
				t.Errorf("entryGrowWork() = %v, want %v", got, tt.want)
			}
			newLen := Size(s)
			if oldLen != newLen {
				t.Errorf("entryGrowWork() len err: %v, want %v", newLen, oldLen)
			}
		})
	}
}

type testOptArg struct {
	name string
	src  *entry
	dst  *entry
	want *entry // must the same type with dst
	max  int
	f    func(old, new *entry)
}

func optConvertTest(t *testing.T, args []testOptArg) {
	for _, arg := range args {
		t.Run(arg.name, func(t *testing.T) {
			arg.f(arg.src, arg.dst)

			sLen, tLen := int(arg.dst.getLen()), int(arg.want.getLen())
			minLen := min(sLen, tLen)
			for i := 0; i < minLen; i++ {
				di := arg.dst.load(i)
				wi := arg.want.load(i)
				if di != wi {
					for j := 0; j < 31; j++ {
						if (di & (1 << j)) != (wi & (1 << j)) {
							t.Errorf("Miss:%d", i*int(arg.dst.bit)+j)
						}
					}
				}
			}
		})
	}
}

func Test_convert(t *testing.T) {
	max := 100000
	u32 := NewOption32(max)
	u31 := NewOption31(max)
	u16 := NewOption16(max)
	u15 := NewOption15(max)

	wu32 := NewOption32(max)
	wu31 := NewOption31(max)
	wu16 := NewOption16(max)
	wu15 := NewOption15(max)

	dst32 := newOptEntry32(uint32(max))
	dst31 := newOptEntry31(uint32(max))
	dst16 := newOptEntry16(uint32(max))
	dst15 := newOptEntry15(uint32(max))

	for i := 0; i < max; i++ {
		u32.Store(uint32(i))
		u31.Store(uint32(i))
		u16.Store(uint32(i))
		u15.Store(uint32(i))

		wu32.Store(uint32(i))
		wu31.Store(uint32(i))
		wu16.Store(uint32(i))
		wu15.Store(uint32(i))
	}
	u31.getEntry().freeze(0)
	u16.getEntry().freeze(0)
	u15.getEntry().freeze(0)
	optConvertTest(t, []testOptArg{
		{
			name: "16->32",
			src:  u16.getEntry(),
			dst:  dst32,
			want: wu32.getEntry(),
			max:  max,
			f:    u16To32,
		},
		{
			name: "15->32",
			src:  u15.getEntry(),
			dst:  dst32,
			want: wu32.getEntry(),
			max:  max,
			f:    u16To32,
		},
		{
			name: "32->16",
			src:  u32.getEntry(),
			dst:  dst16,
			want: wu16.getEntry(),
			max:  max,
			f:    u32To16,
		},
		{
			name: "32->15",
			src:  u32.getEntry(),
			dst:  dst15,
			want: wu15.getEntry(),
			max:  max,
			f:    u32To16,
		},
		{
			name: "32->31",
			src:  u32.getEntry(),
			dst:  dst31,
			want: wu31.getEntry(),
			max:  max,
			f:    u32To31,
		},
		{
			name: "31->32",
			src:  u31.getEntry(),
			dst:  dst32,
			want: wu32.getEntry(),
			max:  max,
			f:    u31To32,
		},
	})
}
