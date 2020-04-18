[![Built with Mage](https://magefile.org/badge.svg)](https://magefile.org)
[![GoDoc](https://godoc.org/github.com/wfscheper/convcom?status.svg)](https://godoc.org/github.com/wfscheper/convcom)
[![Build](https://github.com/wfscheper/convcom/workflows/Build/badge.svg)](https://github.com/wfscheper/convcom/actions?query=workflow%3ABuild)
[![Coverage Status](https://coveralls.io/repos/github/wfscheper/convcom/badge.svg?branch=master)](https://coveralls.io/github/wfscheper/convcom?branch=master)

# convcom

Go library for parsing conventional commits

## Usage

```go
import github.com/wfscheper/convcom
```

Create a config struct to customize parser behavior.

```go
cfg := &Config{}
p := convcom.New(cfg)

commitMessage := `fix(foo): fixed the foos

So many broken foos!

Fixes #7
`

c, err := p.Parse(commitMessage)
if err != nil {
    return err
}
```

See [godoc](https://godoc.org/github.com/wfscheper/convcom) for full details.
