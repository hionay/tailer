# tailer

I was inspired by [samwho/spacer](https://github.com/samwho/spacer), which was written in the Rust, and I really liked it. I decided to write it in the Go, and here we go! :)

tailer is a simple CLI tool to insert lines when command output stops.

![](https://github.com/hionay/tailer/blob/main/images/tailer.gif)

## Installation

```bash
go install github.com/hionay/tailer/cmd/tailer@latest
```

## Usage
Here are the flags you can use with tailer:
```
   --no-color               disable color output (default: false)
   --after value, -a value  duration to wait after last output (default: 1s)
   --dash value, -d value   dash string to print (default: "━")
   --help, -h               show help
```
