# gotype2cli

![ci workflow](https://github.com/pdcalado/gotype2cli/actions/workflows/ci.yml/badge.svg)

Generate code to create a cli command from your Go types.

## What!?

Let's say you've got a type like this:

```go
type Bar struct {
    Height int `json:"height"`
}

// New creates a new bar
func New(height int) Bar {
  return Bar{
    Height: height,
  }
}

// String returns a string representation of the bar
func (b *Bar) String() string {
  return fmt.Sprintf("the bar is %d meters high", +b.Height)
}

// Raise raises the bar by 1
func (b *Bar) Raise() {
  b.Height += 1
}

// RaiseBy raises the bar by the given amount
func (b *Bar) RaiseBy(amount int) {
  b.Height += amount
}

// RaiseFromBars raises the bar by the sum of the heights of the given bars
func (b *Bar) RaiseFromBars(bars ...Bar) {
  for _, bar := range bars {
    b.Height += bar.Height
  }
}
```

Using `gotype2cli` you can generate a cli command that creates and manipulates `Bar` by calling its methods, allowing you to do stuff like:

```console
foo@bar:~$ bar new 10
{"height":10}

foo@bar:~$ # methods with Bar as receiver, read a bar in JSON from stdin
foo@bar:~$ bar new 10 | bar raise
{"height":11}

foo@bar:~$ # methods with Bar as receiver can get a Bar in json as first argument
foo@bar:~$ bar new 10 | bar raise-by 2
{"height":12}

foo@bar:~$ bar new 10 | bar raise | bar raise
{"height":12}

foo@bar:~$ bar new 11 | bar string
"the bar is 11 meters high"

foo@bar:~$ # variadic methods are supported using json arrays
foo@bar:~$ bar new 10 | bar raise-from-bars '[{"height": 2},{"height": 3},{"height": 4}]'
{"height":19}

foo@bar:~$ # exported fields can be set (not implemented)
foo@bar:~$ bar new 10 | bar height 11
{"height":11}

foo@bar:~$ # exported fields can be read (not implemented)
foo@bar:~$ bar new 10 | bar height
{"height":10}

foo@bar:~$ # a patch flag is available to print a JSON patch instead of the object
foo@bar:~$ # (not implemented)
foo@bar:~$ bar new 10 | bar height 11 --patch
[{"op":"replace","path":"/height","value":11}]

foo@bar:~$ # usage and help are generated from code and comments
foo@bar:~$ bar raise-by --help
Usage: bar raise-by [options] <amount>

RaiseBy raises the bar by the given amount.

Returns a modified Bar in json format.
If --patch is set, returns a JSON patch instead.

Bar is read from stdin in JSON format.
```

(function names are converted to kebab-case for a more cli-like experience)

(`--no-object` and `--patch` are not implemented yet, check [TODO](#todo) for more info)

## Gotchas

- generated code depends on [cobra](https://github.com/spf13/cobra)
- methods with Bar as receiver assume first argument is a Bar
- methods returning an error are turned into commands that behave as follows:
  - if error is nil, print Bar in JSON
  - if error is not nil, print error message and exit with status 1
  - if flag `--no-object` is set, print error value (nil or message)
- the flag `--no-object` can be used to disable the default behavior of printing the object
- all types involved must be JSON compatible, including function arguments

## TODO

- [ ] add support for the `--no-object` flag mentioned above.
- [ ] add support for the `--patch` flag mentioned above.
