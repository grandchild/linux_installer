# Linux Installer

A GTK3-based GUI installer for an audience that's used to Windows installers.
When all you would _really_ need is `unzip`, but want a nice user experience
nonetheless. Imitates the look and feel of
[NSIS](https://nsis.sourceforge.io/Screenshots) installers.

![Screenshot](.doc/screenshot_welcome.png)

<sub><sub>
(Example installer [splash image](https://unsplash.com/photos/t1XLQvDqt_4) by
[_Deniz Altindas_](https://unsplash.com/@omeganova) and
[banner image](https://unsplash.com/photos/_TqnKtKCb5w) by
[_Victoria_](https://unsplash.com/@pixeldebris) via [unsplash.com](unsplash.com))
</sub></sub>

A commandline or "*silent*" mode is available as well.


## Contents

* [Requirements](#requirements)
* [Usage](#usage)
  * [Example](#example)
* [Testing](#testing)
* [Customization](#customization)
  * [Images](#images)
  * [Installer Style & Layout](#installer-style-layout)
    * [GUI CSS](#gui-css)
  * [Hooks](#hooks)
  * [New Language Translation](#new-language-translation)
  * [New Installer Screens](#new-installer-screens)
    * [Layout](#layout)
    * [Behavior](#behavior)
* [Hacking](#hacking)
  * [Code Structure](#code-structure)
    * [Main Go Files](#main-go-files)
    * [Helper Go Files](#helper-go-files)
    * [Metadata](#metadata)
    * [Unused Go Files](#unused-go-files)



## Requirements

* Clone the repository anywhere<sup>[(1)](#anchor-1)</sup>:

```bash
    git clone https://github.com/grandchild/linux-installer.git
```

* A working Go *1.11* (or higher) installation, see the
[official installation instructions](https://golang.org/doc/install) for information on
how to install Go on your system

* Make sure your installation works, by following the instructions "*Test your
installation*" on the installation page.

* The `make` & `zip` commands, simply install their packages:
  * `make`
  * `zip`

* On RPM-based systems (Centos, Ubuntu, etc) the following dev-packages need to be
  installed as well:
  * `libgtk-3-dev`
  * `libglib2.0-dev`

* If you want to edit the installer GUI layout you need to install
  [Glade](https://glade.gnome.org/) as well.


## Usage

This project creates an installer *builder* (altough it can create test installers
directly) with which one can create installers for customers.

To create installers with the builder follow these steps

1. Run:<br/>
  `make clean linux-builder.zip`

1. *(Optional)* Copy the `linux-builder.zip` onto the computer you want to create
  installers with.

1. Extract the `linux-builder.zip` in a location of your choice.

1. Add all required files and folders to the contained `data` folder.

1. Edit `resources/config.yml` with any information needed (The version number can be
  set now, or in the next step).

1. Inside the "*linux-builder/*" directory run:<br/>
  `make OUTPUT=<Your installer name here> VERSION=<Your program version here>`

### Example

A command like:

```bash
    make OUTPUT=Setup_ExampleApp_v1.1
```

would create a new file *Setup.bin* which installs the software and shows version 1.1 in
the UI.


## Testing

To simply build and test an installer in the main project, add some files to the `data/`
directory and then run:

```bash
    make clean run
```


## Customization

Various parts of the installer can be customized and changed without touching the code,
such as the style and layout of elements, and the translation strings. The way to do
this is described in this section.

Adding a new screen (or removing one) requires only minor code changes, and is explained
below as well.


### Images

The `splash.bmp` image is shown on the left-hand side of the language-, welcome- and
success-/failure screens, and needs to be a vertical 164×314 pixels in size. The right
side of the image should connect well with the `window.background` color (currently
plain white `#ffffff`).

The `banner.bmp` image is shown on the top of all other screens and needs to be a
horizontal 497×60 pixels in size. The bottom of the image should connect well with the
`window.background` color like above.

The `icon.gif` image is used as the icon in the taskbar and possibly the window's title
bar while the installer is running. It should be a square GIF image, not too large
(16×16 or 32×32 pixels).


### Installer Style & Layout

The GUI layout of the installer is specified in `resources/gui/gui.glade`. This file can
be edited with [Glade](https://glade.gnome.org/), a WYSIWYG GTK3 UI editor.

The layout consists of the main installer window and a do-you-really-want-to-quit dialog
box.

The installer window consists mainly of the "Stack" of screens that the installer can go
through. Not all have to be visited (e.g. installation failure), and not necessarily in
order (although they are mostly run through sequentially).

#### GUI CSS

GTK3 supports styling UI elements with CSS. Elements can have *id*s, *class*es and are
of a type, just like regular HTML elements.

The GUI CSS is loaded from a variable in `resources/config.yml` called `gui_css`, which
controls some colors in the GUI, and mostly sets the background color to white, and
changes some font properties so the license text is not too large, and the filenames in
the installer progress not too prominent.

To change the styling of the installer, simply change the content of that variable.


### Hooks

Before installation starts, and after, there is the possibility of running a custom
script to do any tasks necessary at that time. Currently these are present, but empty.

The hook scripts live in `resources/hooks/` and are named after their execution time,
namely `pre-install.sh` and `post-install.sh`, which run before and after installation,
respectively.

You can write custom commands into these files and they will be executed. Their output
(for debugging purposes) is logged into the installer.log file that is created when the
installer is run.


### New Language Translation

In short: Add a new file named `xx.yml` inside `resources/languages/` (or better, copy
`en.yml`), where "*xx*" is the language's two-letter [ISO
639-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) code. Then translate all
messages. The language should now be available in the language selection in the GUI and
the help text in the CLI mode.

E.g. in order to add French, create and translate `resources/languages/fr.yml`.


### New Installer Screens

A new installer screen is a two-step process:

1. *Layout*: Create a new installer screen in `resources/gui/gui.glade`
1. *Behavior*: Add screen to `screenHandlers()` inside `gui/gui.go`

#### Layout

In the list of screens in Glade, right-click on the screen after which you want to
insert your new screen, and select "*Insert Page After*" at the bottom of the context
menu. Double-click inside the resulting empty space and select GtkBox to create a layout
base for the new screen. Give the Box an ID in the details panel on the right in the
"General" tab. This ID is needed in the next step.

You can then design the inside of the Box however you like. Remember to add IDs to
relevant elements in order to reach them from the behavior code.

#### Behavior

In `gui/gui.go` inside `screenHandlers()` add a new section for the screen like this:

```go
    {
        name: "myscreen",
        before: func() {
            // ...
        },
        after: func() {
            // ...
        },
        undo: func() bool {
            // ...
        }
    },
```

The "*name*" key is the ID you chose in the layout step. The "*before*", "*after*" and
"*undo*" keys are documented in the documentation for the ScreenHandler struct a little
bit earlier in `gui/gui.go`. Refer to that for details.

If a function for a key is empty, the key can be omitted completely.



## Hacking

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


### Code Structure

The code's entry point is the `main()` function in `main/main.go`, which simply calls
the actual main function which is `run.go`'s `Run()`
function[<sup>(2)</sup>](#anchor-2).

All GUI code is placed in its own "main" package, inside the `gui/` folder, because it
is compiled separately as a [Go plugin](https://golang.org/pkg/plugin/) (a Go-specific
kind of *.so* dynamic library). This allows compiling the installer without linking to
GTK3 explicitly, which in turn allows running the installer on older Linux
distributions, and falling back gracefully to an error message and allowing CLI
installation mode. If the code would not be separated out into the plugin, the installer
would always fail on systems without GTK3, reporting only a linker error and without
recourse.


#### Main Go Files

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


#### Helper Go Files

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


#### Metadata

`doc.go` Contains a general description of the installer package, accessible through
[`go doc`](https://golang.org/cmd/doc/) (See above).

`go.mod` & `go.sum` Contain a list of dependencies, their versions and checksums. Used
with [`go mod`](https://golang.org/cmd/mod/).

**Note:** *`go mod` only works if your Go version is* 1.11 *or higher. Use `go version`
to check.*


#### Unused Go Files

`install_windows.go` Roughly implements the functions from `install_linux` but is
largely untested since this installer is not used on Windows, since it would need to be
shipped with GTK3 libraries. Which would defeat the purpose of having a packaged
installer.

`tui.go` A tentative implementation of an ncurses-based terminal-graphical UI for the
installer. Incomplete. The CLI mode serves well enough.



## Footnotes

### (1)
Specifically *not* in `~/go/src/github.com/grandchild/linux_installer`!

### (2)
The reason for the existence of `main/main.go` is that a Go program's main function has
to be placed inside a package called "main". But the installer package with most of the
code should be called "installer", not "main" in order to behave more like an importable
library. Thus the slightly awkward cage for main.
