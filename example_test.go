// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builder_test

import (
	"fmt"

	"github.com/weiwenchen2022/builder"
)

func ExampleBuilder() {
	var b builder.Builder
	for i := 3; i >= 1; i-- {
		_, _ = b.WriteInt(int64(i), 10)
		_, _ = b.WriteString("...")
	}
	_, _ = b.WriteString("ignition")
	fmt.Println(b.String())
	// Output:
	// 3...2...1...ignition
}
