package commands

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NHAS/reverse_ssh/internal/server/webserver"
	"github.com/NHAS/reverse_ssh/internal/terminal"
	"github.com/NHAS/reverse_ssh/internal/terminal/autocomplete"
	"github.com/NHAS/reverse_ssh/pkg/table"
)

type link struct {
}

func (l *link) Run(tty io.ReadWriter, line terminal.ParsedLine) error {

	if line.IsSet("h") {
		return errors.New(l.Help(false))
	}

	if toList, ok := line.Flags["l"]; ok {
		t, _ := table.NewTable("Active Files", "ID", "GOOS", "GOARCH", "Type", "Hits", "Expires", "Path")

		files, err := webserver.List(strings.Join(toList.ArgValues(), " "))
		if err != nil {
			return err
		}

		ids := []string{}
		for id := range files {
			ids = append(ids, id)
		}

		sort.Strings(ids)

		for _, id := range ids {
			file := files[id]

			expiry := "N/A"
			if file.Expiry != 0 {
				expiry = file.Timestamp.Add(file.Expiry).String()
			}
			t.AddValues(id, file.Goos, file.Goarch, file.FileType, fmt.Sprintf("%d", file.Hits), expiry, file.Path)
		}

		t.Fprint(tty)

		return nil

	}

	if toRemove, ok := line.Flags["r"]; ok {
		if len(toRemove.Args) == 0 {
			fmt.Fprintf(tty, "No argument supplied\n")

			return nil
		}

		files, err := webserver.List(strings.Join(toRemove.ArgValues(), " "))
		if err != nil {
			return err
		}

		if len(files) == 0 {
			return errors.New("No links match")
		}

		for id, file := range files {
			err := webserver.Delete(id)
			if err != nil {
				fmt.Fprintf(tty, "Unable to remove %s: %s\n", id, err)
				continue
			}
			fmt.Fprintf(tty, "Removed %s (%s)\n", id, file.Path)
		}

		return nil

	}

	var e time.Duration
	timeStr, err := line.GetArgString("t")
	if err != nil && err != terminal.ErrFlagNotSet {
		return err
	}

	mins, err := strconv.Atoi(timeStr)
	if err != nil {
		return fmt.Errorf("Unable to parse number of minutes (-t): %s", timeStr)
	}
	e = time.Duration(mins) * time.Minute

	goos, err := line.GetArgString("goos")
	if err != nil && err != terminal.ErrFlagNotSet {
		return err
	}

	goarch, err := line.GetArgString("goarch")
	if err != nil && err != terminal.ErrFlagNotSet {
		return err
	}

	homeserver_address, err := line.GetArgString("s")
	if err != nil && err != terminal.ErrFlagNotSet {
		return err
	}

	name, err := line.GetArgString("name")
	if err != nil && err != terminal.ErrFlagNotSet {
		return err
	}

	cc, err := line.GetArgString("cross-compiler")
	if err != nil && err != terminal.ErrFlagNotSet {
		return err
	}

	url, err := webserver.Build(e, goos, goarch, homeserver_address, name, cc, line.IsSet("shared-object"))
	if err != nil {
		return err
	}

	fmt.Fprintln(tty, url)

	return nil
}

func (l *link) Expect(line terminal.ParsedLine) []string {
	if line.Section != nil {
		switch line.Section.Value() {
		case "l", "r":
			return []string{autocomplete.WebServerFileIds}
		}
	}

	return nil
}

func (e *link) Help(explain bool) string {
	if explain {
		return "Generate client binary and return link to it"
	}

	return makeHelpText(
		"link [OPTIONS]",
		"Link will compile a client and serve the resulting binary on a link which is returned.",
		"This requires the web server component has been enabled.",
		"\t-t\tSet number of minutes link exists for (default is one time use)",
		"\t-s\tSet homeserver address, defaults to server --homeserver_address if set, or server listen address if not.",
		"\t-l\tList currently active download links",
		"\t-r\tRemove download link",
		"\t--goos\tSet the target build operating system (default to runtime GOOS)",
		"\t--goarch\tSet the target build architecture (default to runtime GOARCH)",
		"\t--name\tSet link name",
		"\t--shared-object\tGenerate shared object file",
		"\t--cross-compiler\tSpecify C/C++ cross compiler used for compiling shared objects (currently only DLL, linux -> windows)",
	)
}

func Link() *link {
	return &link{}
}
