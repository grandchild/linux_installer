# Hacking

## Contents
* [Code Structure](#code-structure)
  * [Main Go Files](#main-go-files)
  * [Helper Go Files](#helper-go-files)
  * [Metadata](#metadata)
  * [Unused Go Files](#unused-go-files)

This part serves as a general overview over which parts of the code do what. Which
should help if more substantial changes or bug fixes are necessary. For detailed
information on structures and functions, refer to the documentation in the code itself,
which can also be accessed through
[`go doc`](https://golang.org/cmd/go/#hdr-Show_documentation_for_package_or_symbol):

Run
```bash
    go doc --all -u github.com/grandchild/linux_installer
```
and 
```bash
    go doc --all -u github.com/grandchild/linux_installer/gui
```

to see text documentation for all symbols.

To read documentation for a specific item, e.g. the `installFile()` and the
`updateProgressbar()` functions, run:

```bash
    go doc --all -u github.com/grandchild/linux_installer installFile
    go doc --all -u github.com/grandchild/linux_installer/gui updateProgressbar
```


## Code Structure

The code's entry point is the `main()` function in `main/main.go`, which simply calls
the actual main function which is `run.go`'s `Run()`
function[<sup>(1)</sup>](#1).

All GUI code is placed in its own "main" package, inside the `gui/` folder, because it
is compiled separately as a [Go plugin](https://golang.org/pkg/plugin/) (a Go-specific
kind of *.so* dynamic library). This allows compiling the installer without linking to
GTK3 explicitly, which in turn allows running the installer on older Linux
distributions, and falling back gracefully to an error message and allowing CLI
installation mode. If the code would not be separated out into the plugin, the installer
would always fail on systems without GTK3, reporting only a linker error and without
recourse.


### Main Go Files

`run.go` is the entry point and parses commandline flags and decides whether to run in
GUI or CLI mode.

`install.go` provides the `Installer` type that performs the actual installation. It is
used by all installation modes. The installer scans all zips present in the
data_compressed folder, and prepares a list of files to be installed. Once the actual
installation starts, it copies them to the target location on the system. It then
creates an uninstaller script as well as an application menu shortcut, and runs any hook
scripts that have been defined (for either before or after installation).

`install_linux.go` contains the Linux-specific system calls and application-menu,
uninstaller and pre-/post-hooks (which are all OS-specific). It is only compiled when
compiling for Linux (which is what the very first line in the file does).

`gui/gui.go` describes the GUI's behavior. It contains the event handlers at the top,
followed by the constructor. The second half of the code are various functions the GUI
code uses, such as switching from one screen to the next, or checking on the installer's
progress.

`config.go` defines the structure for the config.yml file. It is used throughout the
code, for accessing variables and options.


### Helper Go Files

`resources.go` is the interface around go-rice, the library used for appending data to
the compiled executable—creating the packaged installer. Refer to the [go-rice
documentation](https://github.com/GeertJohan/go.rice) for more details.

`translate.go` implements internationalization functions, and uses language files inside
`resources/languages/` to render any user-facing string in the chosen language of the
installer. It also detects the system locale to allow meaningful user communication
before a language can be chosen.

`variables.go` provides a simple templating engine to expand variables inside strings.
This can even be done recursively (i.e. a variable *value* may contain a variable
reference as well).

`gui/gui_utils.go` contains wrappers for retrieving various GTK3 widget types from the
gui definition file. Go [famously has no generics](https://golang.org/doc/faq#generics),
making this list of similar functions necessary. They all do the same thing, which is
casting a widget loaded from the builder into the desired type.


### Metadata

`doc.go` Contains a general description of the installer package, accessible through
[`go doc`](https://golang.org/cmd/doc/) (See above).

`go.mod` & `go.sum` Contain a list of dependencies, their versions and checksums. Used
with [`go mod`](https://golang.org/cmd/mod/).

**Note:** *`go mod` only works if your Go version is* 1.11 *or higher. Use `go version`
to check.*


### Unused Go Files

`install_windows.go` Roughly implements the functions from `install_linux` but is
largely untested since this installer is not used on Windows, since it would need to be
shipped with GTK3 libraries. Which would defeat the purpose of having a packaged
installer.

`tui.go` A tentative implementation of an ncurses-based terminal-graphical UI for the
installer. Incomplete. The CLI mode serves well enough.


---
#### (1)
The reason for the existence of `main/main.go` is that a Go program's main function has
to be placed inside a package called "main". But the installer package with most of the
code should be called "installer", not "main" in order to behave more like an importable
library. Thus the slightly awkward cage for main.
