# MP3 Joiner

[![GoDoc](https://godoc.org/github.com/jo-hoe/mp3-joiner?status.svg)](https://godoc.org/github.com/jo-hoe/mp3-joiner)
[![Test Status](https://github.com/jo-hoe/mp3-joiner/workflows/test/badge.svg)](https://github.com/jo-hoe/mp3-joiner/actions?workflow=test)
[![Coverage Status](https://coveralls.io/repos/github/jo-hoe/mp3-joiner/badge.svg?branch=main)](https://coveralls.io/github/jo-hoe/mp3-joiner?branch=main)
[![Lint Status](https://github.com/jo-hoe/mp3-joiner/workflows/lint/badge.svg)](https://github.com/jo-hoe/mp3-joiner/actions?workflow=lint)
[![Go Report Card](https://goreportcard.com/badge/github.com/jo-hoe/mp3-joiner)](https://goreportcard.com/report/github.com/jo-hoe/mp3-joiner)

Allow the merge of MP3 files while honoring chapter metadata. This library requires FFmeg to be installed on the target system.

## Requirements 

- [FFmeg](https://ffmpeg.org/download.html)

## Example

```go
package main

import (
 "github.com/jo-hoe/mp3-joiner"
)

func main() {
 builder := NewMP3Builder()
 builder.Append("/path/to/myAudioFile.mp3", 0, 10)
 builder.Append("/path/to/myOtherAudioFile.mp3", 0, -1)
 builder.Build("/path/to/mergedAudioFile.mp3")
}
```

## Development

### Linting

Project used `golangci-lint` for linting.

#### Installation

<https://golangci-lint.run/usage/install/>

#### Execution

Run the linting locally by executing

```cli
golangci-lint run ./...
```

in the working directory

## Further Details

- [How to apply chapters](https://dev.to/montekaka/add-chapter-markers-to-podcast-audio-using-ffmpeg-3c46)
