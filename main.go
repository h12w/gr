// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	flag.Parse()

	// profile
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	app, err := newApp()
	checkError(err)
	defer app.Dispose()

	if args := flag.Args(); len(args) > 0 {
		fileName := args[0]
		app.OpenDoc(fileName)
	}

	checkError(app.Run())
}
