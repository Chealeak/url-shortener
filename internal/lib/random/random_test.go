package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRandomString(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantLen int
	}{
		{name: "size = 1", size: 1, wantLen: 1},
		{name: "size = 5", size: 5, wantLen: 5},
		{name: "size = 10", size: 10, wantLen: 10},
		{name: "size = 20", size: 20, wantLen: 20},
		{name: "size = 30", size: 30, wantLen: 30},
		{name: "size = 0", size: 0, wantLen: 0},
		{name: "size = -1", size: -1, wantLen: 0},
		{name: "size = -10", size: -10, wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str1 := NewRandomString(tt.size)
			str2 := NewRandomString(tt.size)

			assert.Len(t, str1, tt.wantLen)
			assert.Len(t, str2, tt.wantLen)

			if tt.wantLen > 0 {
				assert.NotEqual(t, str1, str2)
			}
		})
	}
}
