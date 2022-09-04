package util

import "testing"

func TestByteCountIEC(t *testing.T) {
	testCases := []struct {
		size uint64
		want string
	}{
		{
			size: 1<<10 - 1,
			want: "1023B",
		},
		{
			size: 1 << 10,
			want: "1.00KB",
		},
		{
			size: 235235,
			want: "229.72KB",
		},
		{
			size: 1<<20 - 1,
			want: "1024.00KB",
		},
		{
			size: 1 << 20,
			want: "1.00MB",
		},
		{
			size: 1 << 30,
			want: "1.00GB",
		},
		{
			size: 1<<64 - 1,
			want: "16.00EB",
		},
	}
	for _, c := range testCases {
		if result := HumanReadableBytes(c.size); result != c.want {
			t.Errorf("actual result is %s but "+
				"want reuslt is %s", result, c.want)
		}
	}
}
