package set

import (
	"testing"
)

func Test_entryGrowWork(t *testing.T) {
	s := NewVarOpt(100)
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
			oldLen := s.Len()
			if got := entryGrowWork(tt.args.s, tt.args.old, tt.args.cap); got != tt.want {
				t.Errorf("entryGrowWork() = %v, want %v", got, tt.want)
			}
			newLen := s.Len()
			if oldLen != newLen {
				t.Errorf("entryGrowWork() len err: %v, want %v", newLen, oldLen)
			}
		})
	}
}
