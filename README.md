# Diagram
[![Build Status](https://travis-ci.org/esimov/diagram.svg?branch=master)](https://travis-ci.org/esimov/diagram)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/esimov/diagram)

Diagram is a CLI tool to generate hand drawn diagrams from ASCII arts. 

It's a full featured CLI application which converts the ASCII text into hand drawn diagrams. The CLI part is based on [gocui](https://github.com/jroimartin/gocui) and the ascii to png conversion is done using the [gg](https://github.com/fogleman/gg) library.

![screencast](images/screencast.gif)

## Installation

In order to run the application please make sure that Go is installed on your local machine and check if `$GOPATH/bin` is included into the `PATH` directory.

```bash
$ export GOPATH="$HOME/go"
$ export PATH="$PATH:$GOPATH/bin"
```
In order to visualize the generated output (with `CTRL-D`) please make sure that [glfw](https://www.glfw.org) is installed on your machine.

A shell script is bundled into the library to mitigate the generation of different binary files for different operating systems, but take care: different dependencies are needed for different operating systems. For a full list of required external dependencies check the official documentation of `go-glfw` (https://github.com/go-gl/glfw/blob/master/README.md).

```bash
$ go get github.com/esimov/diagram
$ go install

# Start the application
$ diagram
```
## Usage

Once you are inside the terminal application you can create, edit or delete the ascii diagrams. By pressing `CTRL+d` you can convert the ASCII art into a handwritten diagram. The `PNG` file will be saved into the `output` folder relative to the current path.

A shell script is included to watch the output folder and automatically open the generated image files.

**Update:**

*This is not needed anymore, since an internal image viewer is bundled into the application.*

### Command Line support

The application also supports the generation of hand drawn diagrams directly from command line without to enter into the CLI application. 

`$ diagram --help` will show the currently supported options:

```bash
Usage of diagram:
  -font string
    	path to font file (default "${GOPATH}/src/github.com/esimov/diagram/font/gloriahallelujah.ttf")
  -in string
    	Source
  -out string
    	Destination
  -preview
    	Show the preview window (default true)
```

#### CLI Examples

Read input from `sample.txt` and write image to `sample.png` showing a preview window with the hand drawn diagram:

```bash
diagram -in sample.txt -out sample.png
```

Read input from `sample.txt` and write image to `sample.png`, and exit immediately without showing a preview window:

```bash
diagram -in sample.txt -out sample.png -preview=false
```

Generate diagram as above but use a font at a different location:

```bash
diagram -in sample.txt -out sample.png -preview=false -font /path/to/my/font/MyHandwriting.ttf
```



### Key bindings
Key                                     | Description
----------------------------------------|---------------------------------------
<kbd>Tab</kbd>                          | Next Panel
<kbd>Shift+Tab</kbd>                    | Previous Panel
<kbd>Ctrl+s</kbd>                       | Open Save Diagram Modal
<kbd>Ctrl+s</kbd>                       | Save Diagram
<kbd>Ctrl+d</kbd>                       | Convert Ascii to PNG
<kbd>Ctrl+x</kbd>                       | Clear the editor content
<kbd>Ctrl+z</kbd>                       | Restore the editor content
<kbd>PageUp</kbd>                       | Jump to the top
<kbd>PageDown</kbd>                     | Jump to the bottom
<kbd>Home</kbd>                         | Jump to the line start
<kbd>End</kbd>                          | Jump to the line end
<kbd>Delete/Backspace</kbd>            | Delete diagram
<kbd>Ctrl+c</kbd>                       | Quit

### Example
| Input | Output |
|:--:|:--:|
| <img src="https://user-images.githubusercontent.com/883386/29396424-9200a978-8320-11e7-9c60-17d2be989136.png" height="300"> | <img src="https://user-images.githubusercontent.com/883386/29396385-529a23a4-8320-11e7-9d70-bf9b33d769cc.png" height="300"> |

The app was tested on **Ubuntu** and **MacOS**.

### Acknowledgements
The ascii to png conversion was ported from [shaky.dart](https://github.com/mraleph/moe-js/blob/master/talks/jsconfeu2012/tools/shaky/web/shaky.dart).

## License

This project is under the MIT License. See the [LICENSE](https://github.com/esimov/diagram/blob/master/LICENSE) file for the full license text.
