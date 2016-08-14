// Copyright ©2016 The mediarename Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "testing"

func TestFileNumberFromPath(t *testing.T) {
	for i, tc := range []struct {
		path string
		want string
	}{
		{"DSCF6165.JPG", "6165"},
		{"VCH_2016-05-24T22.14.54Z_CanonEOS40D_100-7429.jpg", "7429"},
	} {
		got := fileNumberFromPath(tc.path)
		if tc.want != got {
			t.Errorf("Case %v: want %v, got %v", i, tc.want, got)
		}
	}
}
