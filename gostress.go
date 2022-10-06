// The gostress tool runs a `go test` target multiple times.
// It is a drop-in replacement for `go test`.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type logJSON struct {
	Time    time.Time
	Action  string
	Package string
	Test    string
	Output  string
}

type testRunJSON struct {
	attempts int
	success  int
}

var (
	count = flag.Int("count", 1, "times to run the test")
	_     = flag.Bool("failfast", true, "ignored")
	_     = flag.Bool("json", true, "ignored")
)

func main() {
	flag.Parse()
	if len(os.Args) < 2 {
		flag.PrintDefaults()
		return
	}
	args := append([]string{"test"}, fmt.Sprintf("-count=%d", *count), "-failfast", "-json")
	args = append(args, flag.Args()...)
	cmd := exec.Command("go", args...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	read(r)
	cmd.Wait()
}

func read(r io.Reader) {
	runs := make(map[string]*testRunJSON)
	d := json.NewDecoder(r)
	var lastK string
	var lastLen int
	var st time.Time
	var dur time.Duration
	var lastOutput bytes.Buffer
	for {
		var l logJSON
		if err := d.Decode(&l); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
		}
		if l.Test == "" {
			continue
		}
		k := fmt.Sprintf("%v.%v", l.Package, l.Test)
		tr := runs[k]
		if tr == nil {
			tr = &testRunJSON{}
			runs[k] = tr
		}
		switch l.Action {
		case "run":
			st = time.Now()
			tr.attempts++
			continue
		case "pass":
			tr.success++
			lastOutput.Reset()
			dur = time.Since(st)
		case "fail":
			dur = time.Since(st)
		case "output":
			lastOutput.WriteString(l.Output)
			continue
		default:
			continue
		}
		s := fmt.Sprintf("%v: %v/%v %v", k, tr.success, tr.attempts, dur)
		newLen, _ := fmt.Printf("\r%v", s)
		if newLen < lastLen {
			fmt.Print(strings.Repeat(" ", lastLen-newLen))
		}
		if lastK != k {
			if lastK != "" {
				fmt.Println()
				lastLen = 0
			}
			lastK = k
		}
	}
	fmt.Println()
	if lastOutput.Len() > 0 {
		fmt.Println(lastOutput.String())
	}
}
