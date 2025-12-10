package ucheck

import (
	"math/big"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

func Nils(vals ...any) int {
  nils := 0;
  for _, val := range vals {
    if val == nil {
      nils++
      continue
    }
    v := reflect.ValueOf(val)
    switch v.Kind() {
    case reflect.Ptr, reflect.Slice, reflect.Map,
      reflect.Func, reflect.Interface, reflect.Chan:
      if v.IsNil() {
        nils++
      }
    }
  }
  return nils
}

func NonNils(vals ...any) int {
  nonNils := 0;
  for _, val := range vals {
    if val == nil {
      continue
    }
    v := reflect.ValueOf(val)
    switch v.Kind() {
    case reflect.Ptr, reflect.Slice, reflect.Map,
      reflect.Func, reflect.Interface, reflect.Chan:
      if v.IsNil() {
        continue
      }
    }
    nonNils++
  }
  return nonNils
}

func ContainsAll[T comparable](set, query []T) *T {
  var missing T
  for _, q := range query {
    if !slices.Contains(set, q) {
      missing = q
      return &missing
    }
  }
  return nil
}

func ContainsAny[T comparable](set, query []T) *T {
  var found T
  for _, q := range query {
    if slices.Contains(set, q) {
      found = q
      return &found
    }
  }
  return nil
}

type CheckFunc[T any] func(val *T) error

func Check[T any](req *T, checks ...CheckFunc[T]) error {
  for _, check := range checks {
    err := check(req)
    if err != nil {
      return err
    }
  }
  return nil
}

type CheckCountryFunc[T any] func(val *T, country string) error

func CheckCountry[T any](
  req *T, country string, checks ...CheckCountryFunc[T],
) error {
  for _, check := range checks {
    err := check(req, country)
    if err != nil {
      return err
    }
  }
  return nil
}

var reEmail = regexp.MustCompile(`^[-+\w.]{2,50}@[-\w.]{3,30}$`)

func CheckEmail(email string) bool {
  return reEmail.MatchString(email)
}

var reURL = regexp.MustCompile(`^https?://[-\w.:]+(:?/[-\w./%\?=&]*)?$`)

func CheckURL(url string) bool {
  return reURL.MatchString(url)
}

var reIP = regexp.MustCompile(`^(\d{1,3}(?:\.\d{1,3}){3}|[a-f\d:.]{2,39})$`)

func CheckIP(ip string) bool {
  return reIP.MatchString(ip)
}

func CheckPort(port string) bool {
  p, err := strconv.Atoi(port)
  if err != nil {
    return false
  }
  return p >= 1 && p <= 65535
}

var reARN = regexp.MustCompile(`^arn:[-:/\w]{30,}$`)

func CheckARN(arn string) bool {
  return reARN.MatchString(arn)
}

func CheckIDMin(id string, l int) bool {
  return len(strings.Trim(id, " \n")) >= l
}

func CheckIDMinPrefix(id string, l int, prefix string) bool {
  id = strings.Trim(id, " \n")
  return len(id) >= l && strings.HasPrefix(id, prefix)
}

var rePostgresURL = regexp.MustCompile(`^postgres://\w+:\w{40,}@\S+:\d{1,5}/\S+$`)

func CheckPostgresURL(url string) bool {
  return rePostgresURL.MatchString(url)
}

var reIBAN = regexp.MustCompile(`^[A-Z]{2}\d{2}[A-Z0-9]{11,30}$`)

func CheckIBAN(iban, countries string) bool {
  iban = strings.ToUpper(iban)
  if !strings.Contains(strings.ToUpper(countries), iban[:2]) {
    return false
  }
  if !reIBAN.MatchString(iban) {
    return false
  }
  iban = iban[4:] + iban[:4]
  var str strings.Builder
  for _, r := range iban {
    if unicode.IsLetter(r) {
      str.WriteString(strconv.Itoa(int(r - 'A' + 10)))
    } else {
      str.WriteRune(r)
    }
  }
  num := new(big.Int)
  num, valid := num.SetString(str.String(), 10)
  if !valid {
    return false
  }
  return new(big.Int).Mod(num, big.NewInt(97)).Cmp(big.NewInt(1)) == 0
}
