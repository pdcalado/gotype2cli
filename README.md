# gotype2cli

Generate code to create cli command from your Go types.

## What!?

Let's say you've got a type like this:

```go
type Bar struct {
    Height int `json:"height"`
}

func (b *Bar) String() string {
    return "the bar is " + b.Height + " meters high"
}

func (b *Bar) Raise() {
    b.Height += 1
}

// RaiseBy raises the bar by the given amount
func (b *Bar) RaiseBy(amount int) {
    b.Height += amount
}
```

Using `gotype2cli` you can generate a cli command that creates and manipulates `Bar` by calling its methods, allowing you to do stuff like:

```console
foo@bar:~$ bar new --height 10
{"height":10}

foo@bar:~$ # methods with Bar as receiver assume first argument is a Bar
foo@bar:~$ # also assumes Bar is read from stdin if first argument is "-"
foo@bar:~$ bar new --height 10 | bar raise
{"height":11}

foo@bar:~$ # methods with Bar as receiver can get a Bar in json as first argument
foo@bar:~$ bar raise-by '{"height":10}' 1
{"height":11}

foo@bar:~$ bar new --height 10 | bar raise | bar raise
{"height":12}

foo@bar:~$ bar new --height 11 | bar string
the bar is 11 meters high

foo@bar:~$ # exported fields can be set
foo@bar:~$ bar new --height 10 | bar height 11
{"height":11}

foo@bar:~$ # exported fields can be read
foo@bar:~$ bar new --height 10 | bar height
{"height":10}

foo@bar:~$ # a patch flag is available to print a JSON patch instead of the object
foo@bar:~$ bar new --height 10 | bar height 11 --patch
[{"op":"replace","path":"/height","value":11}]

foo@bar:~$ # usage and help are generated from code and comments
foo@bar:~$ bar raise-by --help
Usage: bar raise-by [options] <bar> <amount>

RaiseBy raises the bar by the given amount.

Returns a modified Bar in json format.
If --patch is set, returns a JSON patch instead.

<bar> is a Bar formatted in json or "-" to read from stdin.
```

(function names are converted to kebab-case for a more cli-like experience)

Gotchas:
- methods with Bar as receiver assume first argument is a Bar
- methods returning an error are turned into commands that behave as follows:
  - if error is nil, print Bar in JSON
  - if error is not nil, print error message and exit with status 1
  - if flag `--no-object` is set, print error value (nil or message)
- the flag `--no-object` can be used to disable the default behavior of printing the object
- all types involved must be JSON compatible, including function arguments

