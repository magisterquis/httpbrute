// Program httpbrute brute-forces HTTP paths
package main

/*
 * httpbrute.go
 * Brute-force HTTP paths on an HTTP server
 * By J. Stuart McMurray
 * Created 20200920
 * Last Modified 20200926
 */

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	/* Keep track of the number of tries we've made */
	nTry  int
	nTryL sync.Mutex

	/* Logger which writes to stdout */
	olog = log.New(os.Stdout, "", log.LstdFlags)
)

func main() {
	start := time.Now()
	var (
		target = flag.String(
			"target",
			"",
			"Target `URL`",
		)
		suffixList = flag.String(
			"suffix",
			"",
			"Optional comma-separated `list` of suffixes (e.g. .php,.txt,)",
		)
		wordlist = flag.String(
			"wordlist",
			"",
			"Name of wordlist `file`",
		)
		nPar = flag.Uint(
			"parallel",
			4,
			"Attempt `N` paths in parallel",
		)
		toWait = flag.Duration(
			"timeout-wait",
			10*time.Second,
			"Wait duration before retries after a timeout",
		)
		noLogRetry = flag.Bool(
			"quiet-retries",
			false,
			"Don't log when a URL is retried",
		)
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s [options]

Attempts to brute-force paths on the given target.  If an empty response or
connection timeout occurs during a request, the request will be repeated at
intervals.

Options:
`,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* Parse suffix list */
	suffixes := parseSuffixList(*suffixList)

	/* Make sure we have a target */
	if "" == *target {
		log.Fatalf("Please specify a target with -target")
	}

	/* Open wordlist */
	var wl *os.File
	switch *wordlist {
	case "-": /* stdin */
		wl = os.Stdin
	case "": /* Unsoecified */
		log.Fatalf("Please specify a wordlist with -wordlist")
	default:
		var err error
		wl, err = os.Open(*wordlist)
		if nil != err {
			log.Fatalf(
				"Error opening wordlist file %s: %s",
				*wordlist,
				err,
			)
		}
		defer wl.Close()
	}

	/* Start attackers */
	ch := make(chan string)
	var wg sync.WaitGroup
	for i := uint(0); i < *nPar; i++ {
		wg.Add(1)
		go attack(ch, *toWait, !*noLogRetry, &wg)
	}

	/* Strip / from target */
	*target = strings.TrimRight(*target, "/") + "/"

	/* Tell the user we're about to start */
	log.Printf("Target: %s", *target)
	log.Printf("Suffixes: %q", suffixes)

	/* Send target URLs to attackers */
	scanner := bufio.NewScanner(wl)
	for scanner.Scan() {
		/* Send the word */
		t := *target + scanner.Text()
		/* Send all of the suffixes */
		for _, s := range suffixes {
			ch <- t + s
		}
	}
	if err := scanner.Err(); nil != err {
		log.Printf("Error reading wordlist %s: %s", wl.Name(), err)
	}

	/* Wait for attacks to finish */
	close(ch)
	wg.Wait()
	log.Printf(
		"Tried %d URLs in %s",
		nTry,
		time.Since(start).Round(time.Millisecond),
	)
}

/* parseSuffixList turns a comma-separated list of suffixes into a slice of
unique suffixes */
func parseSuffixList(l string) []string {
	/* Split list into suffixes */
	ps := strings.Split(l, ",")
	/* Uniquify */
	m := make(map[string]struct{})
	for _, s := range ps {
		m[s] = struct{}{}
	}
	ss := make([]string, 0, len(m))
	for s := range m {
		ss = append(ss, s)
	}
	/* Sort the list */
	sort.Strings(ss)
	return ss
}

/* attack reads target URLs from ch and tries to conect to them.  If a timeout
or empty reply happens, attack will wait toWait before trying a again. */
func attack(
	ch <-chan string,
	toWait time.Duration,
	logRetry bool,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	for t := range ch {
		attackOne(t, toWait, logRetry)
		/* Note the attempt */
		nTryL.Lock()
		nTry++
		nTryL.Unlock()
	}
}

/* attackOne attacks a single target.  See attack. */
func attackOne(t string, toWait time.Duration, logRetry bool) {
	/* Only log a retry message once in a row */
	var retryMsg string
	/* Try until we get it */
	for {
		/* Try to get the URL */
		res, err := http.Get(t)
		if nil != err {
			/* maybe print a message and try again in a bit. */
			retryLog(fmt.Sprintf(
				"[Retry] %s Retrying every %s (%s)",
				t,
				toWait,
				err,
			), &retryMsg, toWait, logRetry)
			continue
		}
		defer res.Body.Close()
		/* Don't care about 404's */
		if http.StatusNotFound == res.StatusCode {
			return
		}
		lu, err := res.Location()
		var l string
		if nil != err && !errors.Is(err, http.ErrNoLocation) {
			l = fmt.Sprintf("Error: %s", err)
		} else if !errors.Is(err, http.ErrNoLocation) {
			l = lu.String()
		}
		if "" != l {
			l = fmt.Sprintf(" (Location: %s)", lu)
		}
		/* If we got redirected, report that */
		ru := res.Request.URL.String()
		if ru == t { /* Nothing to report */
			ru = ""
		} else {
			ru = " -> " + ru
		}
		olog.Printf("[%s] %s%s%s", res.Status, t, ru, l)
		break
	}
}

/* retryLog logs m only if *s != m.  If m is logged, it is stored in *s.  As a
convenience, retryLog will wait toWait before returning. */
func retryLog(m string, s *string, toWait time.Duration, logRetry bool) {
	defer time.Sleep(toWait)
	/* Only log the message if we're logging such things */
	if !logRetry {
		return
	}
	/* Already logged it, nothing to do */
	if m == *s {
		return
	}
	/* Log it and save it */
	*s = m
	log.Printf("%s", m)
}
