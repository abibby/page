package calibredb

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	bin         string
	globalFlags *GlobalFlags
	dryRun      bool
}

type GlobalFlags struct {
	// 	Path to the calibre library. Default is to use the path stored in the
	// 	settings. You can also connect to a calibre Content server to perform
	// 	actions on remote libraries. To do so use a URL of the form:
	// 	http://hostname:port/#library_id for example,
	// 	http://localhost:8080/#mylibrary. library_id is the library id of the
	// 	library you want to connect to on the Content server. You can use the
	// 	special library_id value of - to get a list of library ids available on
	// 	the server. For details on how to setup access via a Content server, see
	// 	https://manual.calibre-ebook.com/generated/en/calibredb.html.
	LibraryPath string

	// 	Username for connecting to a calibre Content server
	Username string

	// 	Password for connecting to a calibre Content server. To read the
	// 	password from standard input, use the special value: <stdin>. To read
	// 	the password from a file, use: <f:/path/to/file> (i.e. <f: followed by
	// 	the full path to the file and a trailing >). The angle brackets in the
	// 	above are required, remember to escape them or use quotes for your
	// 	shell.
	Password string

	// 	The timeout, in seconds, when connecting to a calibre library over the
	// 	network. The default is two minutes.
	Timeout time.Duration
}

func (f *GlobalFlags) appendArgs(args []string) []string {
	if f.LibraryPath != "" {
		args = append(args, "--library-path", f.LibraryPath)
	}
	if f.Username != "" {
		args = append(args, "--username", f.Username)
	}
	if f.Password != "" {
		args = append(args, "--password", f.Password)
	}
	if f.Timeout != 0 {
		args = append(args, "--timeout", strconv.Itoa(int(f.Timeout.Seconds())))
	}
	return args
}

func NewClient(bin string, flags *GlobalFlags) *Client {
	c := &Client{
		bin:         bin,
		globalFlags: flags,
	}
	return c
}

type appendArgser interface {
	appendArgs(args []string) []string
}

func (c *Client) exec(ctx context.Context, write bool, flags appendArgser, args ...string) ([]byte, error) {
	staticArgs := []string{}
	if c.globalFlags != nil {
		staticArgs = c.globalFlags.appendArgs(staticArgs)
	}
	staticArgs = append(staticArgs, args...)
	if flags != nil && !reflect.ValueOf(flags).IsNil() {
		staticArgs = flags.appendArgs(staticArgs)
	}

	if write && c.dryRun {
		fmt.Printf("[dry-run] %s %s\n", c.bin, strings.Join(quoteArgs(staticArgs), " "))
		return nil, nil
	}
	b, err := exec.CommandContext(ctx, c.bin, staticArgs...).Output()
	if err != nil {
		return nil, fmt.Errorf("calibredb command failed: %s: %s: %w", c.bin, strings.Join(quoteArgs(staticArgs), " "), err)
	}

	return b, nil
}

func quoteArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t") {
			out[i] = fmt.Sprintf("%q", a)
		} else {
			out[i] = a
		}
	}
	return out
}
