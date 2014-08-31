// Copyright 2014 Simon Zimmermann. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/tideland/goas/v2/monitoring"
	"github.com/simonz05/policy"
	"github.com/simonz05/util/log"
)

var (
	help       = flag.Bool("h", false, "show help text")
	laddr      = flag.String("laddr", ":843", "set bind address for the tcp server")
	version    = flag.Bool("version", false, "show version number and exit")
	cpuprofile = flag.String("debug.cpuprofile", "", "write cpu profile to file")
)

var Version = "0.1.0"

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	log.Println("start policy service …")

	if *version {
		fmt.Fprintln(os.Stdout, Version)
		return
	}

	if *help {
		flag.Usage()
		os.Exit(1)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	err := policy.ListenAndServe(*laddr)

	if err != nil {
		log.Errorln(err)
	}

	monitoring.MeasuringPointsPrintAll()
}
