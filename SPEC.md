# spyglass: CIDR Inspection Utility

**Status:** Architecture frozen. Implementation in progress (driver-loop mode, see Section 1).
**Author of the code:** Paweł (zen66ten). Every line. See "Working Agreement" below.
**Language:** Go, latest stable. Pure standard library. Zero third-party dependencies.
**Module path:** `github.com/zen66ten/spyglass`

---

## 1. Working Agreement (Instructions for the AI Assistant)

This section governs any AI model loaded with this spec during implementation
sessions. It is part of the spec, not commentary.

1. **Paweł writes every line of code himself.** This project exists as a Go
   learning exercise. The model's job is mentorship, not code generation.
2. **Never produce complete implementations of any function in this spec.**
   Not even when asked directly in a moment of frustration. Offer the smallest
   hint that unblocks instead: a relevant stdlib method name, a question that
   exposes the misunderstanding, a pointer to a section of this document or to
   pkg.go.dev.
3. **Code the model may show:** isolated syntax illustrations of a concept not
   tied to spyglass functions (3 to 5 lines max), or Paweł's own code quoted
   back with a specific defect pointed out. Nothing that can be pasted in as a
   solution.
4. **Review mode:** when Paweł shares written code, review it concretely.
   Name the problem, name the line, explain why it is a problem, cite
   Effective Go or the spec (go.dev/ref/spec) where applicable. Do not rewrite
   the function for him. Identify the fix category and let him implement it.
5. **Order of work:** follow Section 9 (Implementation Order). If Paweł jumps
   ahead, flag it once, then help with what he asked.
6. **Learn by building, not book-first.** The Coyle book (*Go Programming From
   Beginner to Professional*, 2nd ed.) is reference material consulted on
   demand during the build, not a prerequisite to clear before starting. When
   a step needs a concept Paweł has not met, name the concept, point him at the
   Coyle chapter or the pkg.go.dev page that covers it, and let him read before
   he writes. Do not pre-teach whole chapters.
7. **Diagnose before explaining.** When Paweł is stuck, first determine
   whether the gap is conceptual, syntactic, a mental model mismatch, or a
   misreading of this spec. Ask what he has tried.
8. **No em dashes. No motivational filler. Cited sources for non-trivial
   claims.**

### Session protocol (the driver loop)

Each build cycle runs in this order. This is the operating mode for every
implementation session, not a one-time setup.

1. **Model states the next unit of work.** One function, one block, or one
   field at a time, in Section 9 order. State what it must do and why it comes
   now. Do not state how.
2. **Paweł proposes an implementation.** In prose, pseudocode, or real Go.
   He may consult the book or pkg.go.dev first and say so.
3. **Model responds in one of three ways:**
   - *Correct:* confirm it, name precisely why it is correct (the rule or
     guarantee that makes it work), then move to step 1 for the next unit.
   - *Wrong or non-idiomatic:* do not give the answer. Point at the specific
     flaw, ask the question that exposes it, or cite the stdlib method or spec
     clause he has not accounted for. Return to step 2.
   - *On the right track but incomplete:* confirm the correct part explicitly
     so he knows what to keep, then narrow to the missing piece.
4. **Repeat until the unit compiles and its test passes**, then advance.

Rules for the loop:
- One unit at a time. Do not queue three functions in one message.
- When Paweł writes working code, he types it into his own files and runs it.
  The model does not hold the codebase; Paweł pastes back what he wrote and the
  compiler or test output when relevant.
- A wrong answer is a teaching opportunity, not a failure to smooth over.
  Name the error plainly.
- If Paweł is stuck after two hints on the same unit, the third hint may be
  larger (a method signature, a 3-to-5-line syntax skeleton of an unrelated
  example), but never the spyglass function itself.

---

## 2. Purpose

A CLI binary that takes an IPv4 CIDR prefix and prints structured information
about it: network address, broadcast, host range, host count, prefix length,
and address scope (public, private, loopback, link-local, CGNAT, multicast,
etc.). Optionally checks whether a given IP address falls inside the prefix.
A verbose mode adds subnet mask, wildcard mask, binary representations, and
classful addressing information.

This is the kind of arithmetic a NOC engineer does mentally or with `ipcalc`.
Building it from scratch in Go teaches structs, methods, multiple return
values, error handling, value semantics, bit manipulation, interfaces via
`io.Writer`, flag parsing, and table-driven testing.

---

## 3. Invocation

```bash
spyglass 192.168.1.0/24
spyglass 192.168.1.0/24 192.168.1.45
spyglass -v 192.168.1.0/24
spyglass --verbose 192.168.1.0/24 192.168.1.45
```

- One positional argument: the CIDR prefix. Required.
- Optional second positional argument: an IP address to test for containment.
- `-v` / `--verbose`: both names bound to the same `*bool` via `flag.FlagSet`.

### Standard output (no flags)

```
Prefix:        192.168.1.0/24
Network:       192.168.1.0
Broadcast:     192.168.1.255
First host:    192.168.1.1
Last host:     192.168.1.254
Total hosts:   254
Prefix length: /24
Scope:         private (RFC 1918)
```

Scope is always shown. It is basic information about a prefix, not extended
information, so it does not live behind the verbose flag.

### Containment check (second positional argument)

Appends one line after the block above:

```
192.168.1.45 is inside 192.168.1.0/24
```

or

```
192.168.2.45 is NOT inside 192.168.1.0/24
```

### Verbose output (`-v` / `--verbose`)

Appends after the standard block:

```
Subnet mask:   255.255.255.0
Wildcard mask: 0.0.0.255
Class:         C (classful range 192.0.0.0 - 223.255.255.255)
Binary (net):  11000000.10101000.00000001.00000000
Binary (mask): 11111111.11111111.11111111.00000000
```

---

## 4. Architecture

### File layout

```
spyglass/
├── go.mod
├── main.go
└── cidr_test.go
```

One package, `main`. No `cmd/`, no `internal/`. That layout exists for
multi-package projects and this is not one.

### Types

```go
// PrefixInfo holds the always-displayed facts about a prefix.
type PrefixInfo struct {
    Prefix     netip.Prefix
    Network    netip.Addr
    Broadcast  netip.Addr
    FirstHost  netip.Addr
    LastHost   netip.Addr
    TotalHosts uint64
    Scope      string
}

// VerboseInfo holds the facts displayed only under -v/--verbose.
type VerboseInfo struct {
    SubnetMask   netip.Addr
    WildcardMask netip.Addr
    Class        string
    BinaryNet    string
    BinaryMask   string
}
```

They are separate structs because they are computed and displayed under
different conditions. Merging them would force every caller to know which
fields are valid when.

### Function signatures (the contract, do not deviate)

```go
func parsePrefix(s string) (netip.Prefix, error)
func prefixInfo(p netip.Prefix) PrefixInfo
func verboseInfo(p netip.Prefix) VerboseInfo
func scope(a netip.Addr) string
func run(args []string, stdout, stderr io.Writer) int
func main()
```

- `parsePrefix` wraps `netip.ParsePrefix`, rejects IPv6 with a clear error,
  and normalizes host bits (input `192.168.1.45/24` yields the masked
  `192.168.1.0/24` via `Prefix.Masked()`).
- `prefixInfo` and `verboseInfo` are pure functions: prefix in, struct out,
  no I/O, no error (they receive an already-validated prefix).
- `scope` classifies an address against well-known ranges (Section 6).
- `run` contains the whole program. It parses flags and arguments, calls the
  pure functions, writes formatted output to the provided writers, and returns
  an exit code. It never calls `os.Exit`.
- `main` is three lines: build the arg slice, call `run` with `os.Stdout` and
  `os.Stderr`, pass the result to `os.Exit`. Anything in `main` is untestable
  because `os.Exit` terminates the test binary; that is the entire reason
  `run` exists.

### Flag parsing

Use `flag.FlagSet`, not the package-level `flag.CommandLine`. A `FlagSet`
constructed inside `run` makes flag parsing testable and re-runnable; the
package-level default is global state. Bind `-v` and `--verbose` to one
`*bool`:

```go
fs := flag.NewFlagSet("spyglass", flag.ContinueOnError)
```

Then two `fs.BoolVar` calls pointing at the same variable. Use
`flag.ContinueOnError` so a parse failure returns an error to handle instead
of exiting. Remaining positional arguments come from `fs.Args()`.

Reference: https://pkg.go.dev/flag#FlagSet

---

## 5. Algorithms (prose only, implementation is the exercise)

### Broadcast address

The broadcast is the network address with all host bits set to 1. Compute the
host mask from the prefix length: for `/n`, the host mask is `2^(32-n) - 1`.
Get the network address as 4 bytes via `Addr.As4()` (this copies; `netip.Addr`
is a value type), convert to a `uint32`, OR the host mask onto it, convert
back. `encoding/binary` has helpers, or shift bytes manually; either is
acceptable in v1.

### Subnet mask and wildcard mask

Subnet mask for `/n` is a `uint32` with the top `n` bits set:
`^uint32(0) << (32 - n)`. The `/0` case needs care: a shift count of 32 on a
uint32 is legal in Go (unlike C) and yields 0, which is correct here, but
verify it rather than assuming. Wildcard mask is the bitwise NOT of the
subnet mask.

Spec reference for shift behavior: go.dev/ref/spec#Operators ("if the left
operand is an unsigned integer, shifts larger than the width produce 0").

### Binary formatting

`fmt.Sprintf("%08b", b)` formats one byte as 8 binary digits, zero-padded.
Join four of them with dots. Reference: https://pkg.go.dev/fmt (verb `%b`,
width and zero-padding flags).

### Host counting

For `/n` with n <= 30: `TotalHosts = 2^(32-n) - 2`. Special cases in
Section 7.

---

## 6. Scope Classification

`scope` checks the address against well-known ranges in order and returns the
first match. Define the ranges as package-level variables using
`netip.MustParsePrefix` (panicking on a malformed literal at init is correct
behavior for compile-time-known constants).

| Range            | Scope string                  | Reference |
|------------------|-------------------------------|-----------|
| 10.0.0.0/8       | private (RFC 1918)            | RFC 1918  |
| 172.16.0.0/12    | private (RFC 1918)            | RFC 1918  |
| 192.168.0.0/16   | private (RFC 1918)            | RFC 1918  |
| 100.64.0.0/10    | shared / CGNAT (RFC 6598)     | RFC 6598  |
| 127.0.0.0/8      | loopback (RFC 1122)           | RFC 1122  |
| 169.254.0.0/16   | link-local (RFC 3927)         | RFC 3927  |
| 192.0.2.0/24     | documentation (RFC 5737)      | RFC 5737  |
| 198.51.100.0/24  | documentation (RFC 5737)      | RFC 5737  |
| 203.0.113.0/24   | documentation (RFC 5737)      | RFC 5737  |
| 224.0.0.0/4      | multicast (RFC 5771)          | RFC 5771  |
| 240.0.0.0/4      | reserved (RFC 1112)           | RFC 1112  |
| anything else    | public                        |           |

Order matters only where ranges could overlap; with this set they do not, but
iterate a slice in declared order anyway so the behavior is deterministic and
extensible.

Note: `netip.Addr` has built-in helpers (`IsPrivate`, `IsLoopback`,
`IsLinkLocalUnicast`, `IsMulticast`). Do not use them for the main
classification; the point of this function is writing the range table and the
containment loop yourself. Mentioning them in a code comment as the
"production" alternative is encouraged.

---

## 7. Special Cases

| Prefix | Behavior |
|--------|----------|
| /32    | Host route. Network = Broadcast = FirstHost = LastHost = the address. TotalHosts = 1. |
| /31    | Point-to-point (RFC 3021). No broadcast concept. TotalHosts = 2. FirstHost = network address, LastHost = the other address. Broadcast field set to the zero value `netip.Addr{}`; the print logic skips the Broadcast line when it is not valid (`Addr.IsValid()`). |
| /0     | Valid input. TotalHosts = 2^32 - 2 = 4294967294. This is why TotalHosts is `uint64`, not `int` or `uint32`: the intermediate `2^32` does not fit in 32 bits. |

A `switch p.Bits()` block handling /31 and /32 before the general path is
sufficient. Do not over-engineer.

### Classful addressing (verbose only)

| Class | First-octet range | Notes                  |
|-------|-------------------|------------------------|
| A     | 0 - 127           |                        |
| B     | 128 - 191         |                        |
| C     | 192 - 223         |                        |
| D     | 224 - 239         | multicast              |
| E     | 240 - 255         | reserved               |

Classful addressing is historical (obsoleted by CIDR, RFC 4632) which is why
it lives in verbose output, not the default block.

---

## 8. Error Handling Contract

All errors go to stderr. `run` returns the exit code; `main` passes it to
`os.Exit`.

| Condition                          | Behavior                          | Exit |
|------------------------------------|-----------------------------------|------|
| No positional arguments            | usage message to stderr           | 1    |
| More than two positional arguments | usage message to stderr           | 1    |
| Invalid CIDR string                | parse error to stderr             | 1    |
| IPv6 prefix                        | "IPv6 not supported in v1"        | 1    |
| Second arg not a valid IPv4 addr   | parse error to stderr             | 1    |
| Flag parse failure                 | flag package error (stderr)       | 1    |
| Valid input                        | output to stdout                  | 0    |

---

## 9. Implementation Order

Work in this order. Each step compiles and is tested before the next begins.

1. `go mod init github.com/zen66ten/spyglass`. Empty `main` that compiles.
2. `parsePrefix` + its table-driven tests (valid input, host bits set,
   garbage string, IPv6 rejection).
3. `prefixInfo` for the general case (/1 through /30) + tests from the
   Section 10 table.
4. Special cases /31, /32, /0 + tests.
5. `scope` + tests (one address per range, plus a public one).
6. `run` happy path: parse args manually first (no flags yet), print the
   standard block. Tests against `bytes.Buffer`.
7. Containment check (second positional arg).
8. `flag.FlagSet` integration, `-v`/`--verbose`.
9. `verboseInfo` + tests (masks, binary strings, class).
10. Error-path tests for `run` (Section 8 table).

---

## 10. Test Cases

Table-driven tests, a slice of structs, one `t.Run` subtest per entry. Coyle
covers the pattern in the testing chapter.

### prefixInfo

```
Input CIDR          Network        Broadcast       FirstHost      LastHost        TotalHosts  Bits
192.168.1.0/24      192.168.1.0    192.168.1.255   192.168.1.1    192.168.1.254   254         24
10.0.0.0/8          10.0.0.0       10.255.255.255  10.0.0.1       10.255.255.254  16777214    8
172.16.0.0/12       172.16.0.0     172.31.255.255  172.16.0.1     172.31.255.254  1048574     12
192.168.1.0/30      192.168.1.0    192.168.1.3     192.168.1.1    192.168.1.2     2           30
192.168.1.0/31      192.168.1.0    (invalid)       192.168.1.0    192.168.1.1     2           31
10.0.0.1/32         10.0.0.1       10.0.0.1        10.0.0.1       10.0.0.1        1           32
```

### parsePrefix

- `"192.168.1.45/24"` returns the masked `192.168.1.0/24`, no error
- `"notacidr"` returns an error
- `"2001:db8::/32"` returns an error (IPv6 rejection)
- `"10.0.0.0/33"` returns an error

### scope

One test case per row of the Section 6 table, plus `8.8.8.8` returning
`public`.

### verboseInfo

```
Input CIDR        SubnetMask       WildcardMask   Class
192.168.1.0/24    255.255.255.0    0.0.0.255      C
10.0.0.0/8        255.0.0.0        0.255.255.255  A
172.16.0.0/12     255.240.0.0      0.15.255.255   B
224.0.0.0/4       240.0.0.0        15.255.255.255 D
```

Plus binary string checks for at least 192.168.1.0/24.

### run

- No args: exit code != 0, usage text in the stderr buffer
- Valid CIDR only: exit 0, "Prefix:" and "Scope:" in stdout buffer
- Valid CIDR + contained IP: exit 0, "is inside" in stdout
- Valid CIDR + non-contained IP: exit 0, "is NOT inside" in stdout
- Invalid CIDR: exit 1, error text in stderr
- `-v` with valid CIDR: exit 0, "Subnet mask:" in stdout

---

## 11. Imports

```go
// main.go
import (
    "flag"
    "fmt"
    "io"
    "net/netip"
    "os"
)

// cidr_test.go
import (
    "bytes"
    "net/netip"
    "strings"
    "testing"
)
```

No third-party packages. Reaching for `github.com/anything` means the design
has drifted; stop and re-read this spec.

---

## 12. Build and Run

```bash
go mod init github.com/zen66ten/spyglass   # once
go test -v ./...
go vet ./...
go build -o spyglass .
./spyglass 192.168.1.0/24
./spyglass -v 192.168.1.0/24 192.168.1.45
```

---

## 13. What This Teaches

| Concept               | Where it appears                                          |
|-----------------------|-----------------------------------------------------------|
| Structs               | PrefixInfo, VerboseInfo definitions                       |
| Methods               | netip.Prefix and netip.Addr method calls                  |
| Multiple return       | parsePrefix returns (netip.Prefix, error)                 |
| Error handling        | Every parse call, IPv6 check, argument validation         |
| Value types           | netip.Addr is a value, not a pointer; As4() copies        |
| Bit manipulation      | Broadcast, subnet mask, wildcard mask                     |
| fmt verbs             | %08b binary formatting, width/padding flags               |
| flag.FlagSet          | Testable flag parsing; -v and --verbose share one Bool    |
| io.Writer interface   | run() accepts writers; this is what makes the CLI testable|
| Package-level vars    | MustParsePrefix for the scope range table                 |
| Table-driven tests    | Every function in cidr_test.go                            |
| os.Exit semantics     | Why run() exists and main() is three lines                |

---

## 14. Explicit Non-Goals for v1

- IPv6 support
- JSON output
- Reading prefixes or hosts from a file
- CIDR overlap detection
- Concurrency of any kind
- Any third-party dependency

---

## 15. Version History

| Version | Status   | Description                                          |
|---------|----------|------------------------------------------------------|
| v1      | this doc | Single IPv4 CIDR, scope always shown, -v verbose, optional containment check |
| v2      | future   | File input, multiple CIDRs, overlap detection        |
| v3      | future   | Concurrent reachability check, context cancellation  |
