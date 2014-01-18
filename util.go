// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func p(v ...interface{}) {
	fmt.Println(v...)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func lines(s ...string) string {
	return strings.Join(s, "\n")
}

func round(f float32) int {
	return int(f + 0.5)
}

func sqrtf(f float32) float32 {
	return float32(math.Sqrt(float64(f)))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func mt(f func()) float64 {
	t := time.Now()
	f()
	du := time.Now().Sub(t)
	return du.Seconds()
}
