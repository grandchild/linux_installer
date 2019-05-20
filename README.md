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


## Requirements

* A working Go installation, see the
[official installation instructions](https://golang.org/doc/install) for information on
how to install Go on your system
* Make sure your installation works, by following the instructions "*Test your
installation*" on the installation page.
* The repository needs to be cloned into
   $GOPATH/src/github.com/grandchild/linux_installer
* Additional packages needed to build (linux-builder):
    *   make
    *   libgio2.0-cil-dev
    *   libglib2.0-dev
    *   libcairo2-dev
    *   libgdk3.0-cil-dev
    *   libgtk-3-dev
    *   libpango1.0-dev
    *   golang-rice **
* Additional packages needed to execute linux-builder:
    *   zip

** Copying rice binary not working, Makefile has to be adjustet to point to /usr/bin/rice instead of $GOPATH/bin/rice.
    Alternatively, rice can be build manually.


## Usage

This project creates an installer *builder* (altough it can create test installers
directly) with which one can create installers for customers.

To create the builder run

```bash
    make clean linux-builder.tar.gz
```

You can then copy and extract the linux-builder onto another system, fill the contained
data folder with files, edit the `config.yml` with any information needed, and inside
the "*linux-builder/*" directory run

```bash
    make OUTPUT=<Your installer name here> VERSION=<Your program version here>
```

So for example something like

```bash
    make OUTPUT=Setup_ExampleApp_v1.1
```

would create a new file *Setup.bin* which installs the software.

## Testing

To simply build and test an installer in the main project, run
```bash
    make clean run
```


## Hacking

### Code Structure
TODO

### Installer style & layout
TODO

### New installer screens
TODO
