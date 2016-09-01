package main

import (
	"os"
	"runtime/pprof"

	log "github.com/Sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/uswitch/loglet"
	"github.com/uswitch/loglet/cmd/loglet/options"
)

func main() {
	l := options.NewLoglet()
	l.AddFlags()

	kingpin.Parse()

	if l.CpuProfile != "" {
		f, err := os.Create(l.CpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	log.SetLevel(l.LogLevel)
	log.Infof("starting")

	err := loglet.Run(l)
	if err != nil {
		log.Errorf("%s", err)
		os.Exit(1)
	}

	log.Infof("done")
}
