# tailer

[![Go](https://github.com/hionay/tailer/actions/workflows/go.yml/badge.svg)](https://github.com/hionay/tailer/actions/workflows/go.yml)

I was inspired by [samwho/spacer](https://github.com/samwho/spacer), which was written in Rust, and I really liked it. I decided to write it in Go, and here we go! :)

`tailer` is a simple CLI tool to insert lines when command output stops.

![](https://github.com/hionay/tailer/blob/main/images/tailer.gif)

## Installation

```bash
go install github.com/hionay/tailer/cmd/tailer@latest
```

## Usage
Here are the commands and flags you can use with `tailer`:
```
COMMANDS:
   exec, e  execute a command and tail its output
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --no-color               disable color output (default: false)
   --after value, -a value  duration to wait after last output (default: 1s)
   --dash value, -d value   dash character to print (default: "─")
   --help, -h               show help
   --version, -v            print the version
```

You can use `tailer` to execute a command with `exec` without the need for piping.

```bash
$ tailer exec "python3 -m http.server 9300"
```
