~Please note that httpbrute is still in active development and the interface is subject to change.~
===================================================================================================

HTTP Bruteforcer
================
HTTPBrute is a simple HTTP path brute-forcer written to look fairly un-hacker
while still enumerating URLs.

Features:
- Thin wrapper around the commonly-used [Go HTTP library](https://golang.org/pkg/net/http/)
- Retries requests after timeouts and other network errors
- Follows redirects and reports both the found and final URL
- Can add arbitrary suffixes (e.g. `.php`) to URLs
- Automatic HTTP/2 support

Please run with `-h` for a complete list of configurable options.

For legal use only.

Quickstart
----------
```sh
go get github.com/magisterquis/httpbrute
go install github.com/magisterquis/httpbrute
httpbrute -h
httpbrute -target https://example.com -wordlist ./wordlist
```

Wordlists
---------
Wordlists are specified with `-wordlist` and should contain one path suffix per
line, for example:

```
index
login
.git
.well-known
.ssh
phpinfo
wp-admin
```

The wordlists which come with [dirb](http://dirb.sourceforge.net/) and other
HTTP brute-forcers work just fine.

A wordlist can be read from stdin with `-wordlist -`, e.g.
```sh
./wordlist-generator | httpbrute -target https://example.com -wordlist - 
```

Suffixes
--------
Suffixes may be added to each entry in the wordlist by specifying the suffixes
in a comma-separated list with `-suffix`.  If suffixless queries are also
desired, the list may be terminated in a comma to indicate an empty suffix.

Example: Make queries for `.php`, `.txt`, and no suffix:
```sh
httpbrute -target https://example.com -wordlist ./wordlist -suffix .php,.txt,
```

Target Specification
--------------------
The target base URL is specified with `-target`.  To the target will be
appended each line of the wordlist.  Targets may end with a `/`; if not one
will be silently added.

The example wordlist in the [Wordlists](#wordlists) section with
`-target https://example.com` would result in queries for
`https://example.com/index`, `https://example.com/login`, and so on.

Parallelism
-----------
A fairly large number of HTTP requests can be made in parallel, controlled with
`-parallel`.  Setting this too high can cause problems to underpowered
webservers.  This should be avoided.  The practial upper limit is probably
somewhere around `ulimit -n`, though it may be less (because stdio) or more
(because HTTP/2).

Output
------
Paths which returned a non-404 status will be logged to stdout.  Everything
else goes to stderr.  Something like the following is probably not a bad idea:

```sh
httpbrute <flags> | tee httpbrute.out
```
