// diff renders a diff in a local browser, suitable for use on crostini.
package main

import (
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"sourcegraph.com/sourcegraph/go-diff/diff"
)

const css = `
section {
  border: solid 1px #aaa;
  margin-bottom: 2em;
}
section h2 {
  font-weight: normal;
  font-size: 100%;
  background: #eee;
  margin: 0;
  padding: 0.5ex 1ex;
}
.hunk {
  border-spacing: 0;
  font-family: monospace;
  white-space: pre-wrap;
  line-height: 1.3em;
}
.hunk td {
  vertical-align: top;
}
.del {
  background: #fcc;
}
.add {
  background: #cfc;
}
`

func renderFile(w io.Writer, f *diff.FileDiff) {
	fmt.Fprintf(w, "<section>")
	fmt.Fprintf(w, "<h2>%s</h2>", f.OrigName)

	for hi, h := range f.Hunks {
		if hi > 0 {
			fmt.Fprintf(w, "<hr>")
		}
		fmt.Fprintf(w, "<!-- %s -->", html.EscapeString(fmt.Sprintf("%#v", h)))
		fmt.Fprintf(w, "<table class=hunk width='100%%'>")
		left := []string{}
		right := []string{}
		lines := strings.Split(string(h.Body), "\n")
		for _, l := range lines {
			line := html.EscapeString(string(l))
			ltype := ' '
			if len(line) > 0 {
				ltype = rune(line[0])
				line = line[1:]
			}
			switch ltype {
			case '-':
				left = append(left, fmt.Sprintf("<td class=del>- %s</td>", line))
			case '+':
				right = append(right, fmt.Sprintf("<td class=add>+ %s</td>", line))
			case ' ':
				for len(left) < len(right) {
					left = append(left, "<td></td>")
				}
				for len(right) < len(left) {
					right = append(right, "<td></td>")
				}
				left = append(left, fmt.Sprintf("<td> %s</td>", line))
				right = append(right, fmt.Sprintf("<td> %s</td>", line))
			}
		}
		for len(left) < len(right) {
			left = append(left, "<td></td>")
		}
		for len(right) < len(left) {
			right = append(right, "<td></td>")
		}
		for i := 0; i < len(left); i++ {
			fmt.Fprintf(w, "<tr>%s%s</tr>", left[i], right[i])
		}
		fmt.Fprintf(w, "</table>")
	}
	fmt.Fprintf(w, "</section>")
}

func render(w io.Writer, diffs []*diff.FileDiff) {
	fmt.Fprintf(w, "<style>%s</style>", css)
	for _, f := range diffs {
		renderFile(w, f)
	}
}

func serve(diffs []*diff.FileDiff) error {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("http://localhost:%d", port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			return
		}
		render(w, diffs)
	})

	fmt.Printf("spawning browser on %s\n", addr)
	exec.Command("www-browser", addr).Start()

	return http.Serve(listener, nil)
}

func run(args []string) error {
	var r io.Reader = os.Stdin
	if len(args) > 0 {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}

	diffs, err := diff.NewMultiFileDiffReader(r).ReadAllFiles()
	if err != nil {
		return err
	}
	return serve(diffs)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
