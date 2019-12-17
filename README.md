# Importsort Utility for Go projects

This project is based on [goarista/importsort](https://github.com/aristanetworks/goarista/tree/master/cmd/importsort).

**Why another tool**
 
Neither `goimports` nor `gofmt` group go files like recommended in the [golang documentation](https://github.com/golang/go/wiki/CodeReviewComments#imports).
The documentation suggests a three-group pattern: 
- standard library
- local imports, 
- then third-party imports 

```go
import (
	"fmt"
	"hash/adler32"
	"os"

	"appengine/foo"
	"appengine/user"

	"github.com/foo/bar"
	"rsc.io/goversion/version"
)
```

To the contrary, the popular tool `goimports` groups the imports in the following (different) order:
- standard library
- third-party imports 
- then local imports


```bash
goimports -local "<my vcs root directory" <myDirectory>
goimports -local "gardener/gardener" pkg
```

```go
import (
	"fmt"
	"hash/adler32"
	"os"
	
	"github.com/foo/bar"
	"rsc.io/goversion/version"
	
	"appengine/foo"
	"appengine/user"
)
```

## How to use

```bash
importsort -help
Usage of importsort:
  -exclude string exclude filenames matching this regex pattern
  -l	   list files whose formatting differs from importsort
  -s       prefix package prefix to define an import section, ex: "cvshub.com/company". May be specified multiple times. If not specified the repository root is used.
  -w	   write result to file instead of stdout
```
If -s is not provided, the tool automatically figures out the vcs root path for you. This is equivalent to the -local flag in goimports.

**Example:**
- prints all files that are not properly grouped in the directories pkg and cmd
- ignores files that have the substring zz_generated inside
```bash
importsort -exclude zz_generated -l pkg cmd 
```