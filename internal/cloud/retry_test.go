// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	tests := []struct {
		name string
		want time.Duration
		env  string
	}{
		{
			name: "env value set to 1m",
			want: 1 * time.Minute,
			env:  "1m",
		},
		{
			name: "env value is not set",
			want: defaultTimeoutDuration,
			env:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tfMaxTimeout, tt.env)
			once = new(sync.Once)
			if got := Timeout(); got != tt.want {
				t.Errorf("Timeout() = %v, want %v", got, tt.want)
			}
		})
	}
}
