package uquery

import (
	"regexp"
	"strings"
	"time"

	"github.com/volodymyrprokopyuk/go-util/urand"
	"github.com/volodymyrprokopyuk/go-util/userv"
)

var reSQLState = regexp.MustCompile(` \(SQLSTATE .+\)`)

func HTTPError(err error) error {
  if err != nil {
    msg := err.Error()
    if strings.Contains(msg, "ck: ") {
      msg = strings.ReplaceAll(msg, "ERROR: ", "")
      msg = strings.ReplaceAll(msg, "ck: ", "")
      msg = reSQLState.ReplaceAllString(msg, "")
      return userv.BadRequest(msg)
    }
  }
  return err
}

func Retry(query func() error, times int, failure string) error {
  var err error
  for range times {
    err = query()
    if err != nil &&
      strings.Contains(err.Error(), failure) {
      time.Sleep(time.Duration(urand.RandInt(500, 800)) * time.Millisecond)
      continue
    } else {
      break
    }
  }
  return err
}
