package set

import (
	"reflect"
	"testing"
)

func TestSort(t *testing.T) {

	type args struct {
		array []int32
	}
	tests := []struct {
		name string
		args args
		want []int32
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				array: []int32{0, -2, -1, 1, 1, 2, 1, 1, 3, 2, 1, 5, 2, 1, 1, 3, 5, 1, 2, 3},
			},
			want: []int32{-2, -1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 5, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if Sort(tt.args.array); !reflect.DeepEqual(tt.args.array, tt.want) {
				t.Errorf("Sort() = %v, want %v", tt.args.array, tt.want)
			}
		})
	}
}
