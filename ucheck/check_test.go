package ucheck_test

import (
	"strings"
	"testing"

	"github.com/volodymyrprokopyuk/go-util/ucheck"
	"github.com/volodymyrprokopyuk/go-util/urand"
)

func TestCheckNilsNonNilsSuccess(t *testing.T) {
  type typ struct {
    a *int
    b *string
  }
  a, b := 1, "a"
  var t1 typ
  t2 := typ{a: &a}
  t3 := typ{a: &a, b: &b}
  cases := []struct{
    name string
    vals []any
    nils int
  }{
    {"values", []any{nil, 1, nil, "a"}, 2},
    {"types 1", []any{t1.a, t1.b}, 2},
    {"types 2", []any{t2.a, t2.b}, 1},
    {"types 3", []any{t3.a, t3.b}, 0},
    {"values and types", []any{nil, 1, nil, "a", t2.a, t2.b}, 3},
  }
  for _, c := range cases {
    t.Run(c.name, func(t *testing.T) {
      nils := ucheck.Nils(c.vals...)
      nonnils := ucheck.NonNils(c.vals...)
      if nils != c.nils {
        t.Errorf("expected %d, got %d", c.nils, nils)
      }
      exp := len(c.vals) - nils
      if nonnils != exp {
        t.Errorf("expected %d, got %d", exp, nonnils)
      }
    })
  }
}

func TestCheckContainsAllSuccess(t *testing.T) {
  mis1, mis4 := 1, 4
  cases := []struct{
    name string
    set []int
    query []int
    missing *int
  }{
    {"empty set query", []int{}, []int{}, nil},
    {"empty set", []int{}, []int{1, 2}, &mis1},
    {"empty query", []int{1, 2}, []int{}, nil},
    {"contains all", []int{1, 2, 3, 4}, []int{1, 2, 3}, nil},
    {"missing", []int{1, 2, 3}, []int{1, 2, 3, mis4}, &mis4},
  }
  for _, c := range cases {
    t.Run(c.name, func(t *testing.T) {
      missing := ucheck.ContainsAll(c.set, c.query)
      if missing == nil {
        if missing != c.missing {
          t.Errorf("expected %v, got %v", *c.missing, missing)
        }
      } else {
        if c.missing == nil {
          t.Errorf("expected %v, got %v", c.missing, *missing)
        } else if *missing != *c.missing {
          t.Errorf("expected %v, got %v", *c.missing, *missing)
        }
      }
    })
  }
}

func TestCheckContainsAnySuccess(t *testing.T) {
  fnd1 := 1
  cases := []struct{
    name string
    set []int
    query []int
    found *int
  }{
    {"empty set query", []int{}, []int{}, nil},
    {"empty set", []int{}, []int{1, 2}, nil},
    {"empty query", []int{1, 2}, []int{}, nil},
    {"contains any", []int{1, 2, 3, 4}, []int{1, 2, 3}, &fnd1},
    {"contains none", []int{1, 2, 3}, []int{4, 5}, nil},
  }
  for _, c := range cases {
    t.Run(c.name, func(t *testing.T) {
      found := ucheck.ContainsAny(c.set, c.query)
      if found == nil {
        if found != c.found {
          t.Errorf("expected %v, got %v", *c.found, found)
        }
      } else {
        if c.found == nil {
          t.Errorf("expected %v, got %v", c.found, *found)
        } else if *found != *c.found {
          t.Errorf("expected %v, got %v", *c.found, *found)
        }
      }
    })
  }
}

func TestCheckEmailSuccess(t *testing.T) {
  email := urand.RandEmail()
  if !ucheck.CheckEmail(email) {
    t.Errorf("invalid email: %s", email)
  }
}

func TestCheckURLSuccess(t *testing.T) {
  url := urand.RandURL()
  if !ucheck.CheckURL(url) {
    t.Errorf("invalid URL: %s", url)
  }
}

func TestCheckIPSuccess(t *testing.T) {
  ip := urand.RandIP()
  if !ucheck.CheckIP(ip) {
    t.Errorf("invalid IP: %s", ip)
  }
  ip = urand.RandIPv6()
  if !ucheck.CheckIP(ip) {
    t.Errorf("invalid IPv6: %s", ip)
  }
}

func TestCheckIBANSuccessFailure(t *testing.T) {
  country := "es"
  for range 2 {
    iban := urand.RandIBAN(country)
    if !ucheck.CheckIBAN(iban, country) {
      t.Errorf("invalid IBAN %s", iban)
    }
  }
  for range 2 {
    iban := urand.RandIBAN(country)
    invalid := strings.ReplaceAll(iban, iban[2:4], urand.Rand123(2))
    if ucheck.CheckIBAN(invalid, country) {
      t.Logf("invalid IBAN %s has passed check", invalid)
    }
  }
}
