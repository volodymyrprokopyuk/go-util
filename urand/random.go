package urand

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math/big"
	"strings"
	"time"
)

func intP(i int) *int {
  return &i
}

func stringP(s string) *string {
  return &s
}

func timeP(t time.Time) *time.Time {
  return &t
}

func RandInt(a, b int) int {
  lim := big.NewInt(int64(b - a))
  rnd, _ := rand.Int(rand.Reader, lim)
  res := rnd.Int64() + int64(a)
  return int(res)
}

func RandIntP(a, b int) *int {
  return intP(RandInt(a, b))
}

var (
  alpha = strings.Split("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", "")
  lower = strings.Split("abcdefghijklmnopqrstuvwxyz", "")
  upper = strings.Split("ABCDEFGHIJKLMNOPQRSTUVWXYZ", "")
  digit = strings.Split("0123456789", "")
  hex = strings.Split("0123456789abcdef", "")
  alnum = strings.Split(
    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", "",
  )
  symbol = strings.Split("!@#$%^&*_+", "")
  alnumsym = strings.Split(
    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*_+", "",
  )
)

func RandBytes(l int) []byte {
  rnd := make([]byte, l)
  _, _ = rand.Read(rnd)
  return rnd
}

func RandAbc(l int) string {
  rnd := make([]byte, l)
  _, _ = rand.Read(rnd)
  var bld strings.Builder
  for _, b := range rnd {
    bld.WriteString(alpha[int(b) % len(alpha)])
  }
  return bld.String()
}

func RandAbcP(l int) *string {
  return stringP(RandAbc(l))
}

func Rand123(l int) string {
  rnd := make([]byte, l)
  _, _ = rand.Read(rnd)
  var bld strings.Builder
  for _, b := range rnd {
    bld.WriteString(digit[int(b) % len(digit)])
  }
  return bld.String()
}

func Rand123P(l int) *string {
  return stringP(Rand123(l))
}

func RandHex(l int) string {
  var bld strings.Builder
  for range l {
    bld.WriteString(RandFrom(hex...))
  }
  return bld.String()
}

func RandHexP(l int) *string {
  return stringP(RandHex(l))
}

func RandStr(l int) string {
  rnd := make([]byte, l)
  _, _ = rand.Read(rnd)
  var bld strings.Builder
  for _, b := range rnd {
    bld.WriteString(alnum[int(b) % len(alnum)])
  }
  return bld.String()
}

func RandStrP(l int) *string {
  return stringP(RandStr(l))
}

func RandSym(l int) string {
  rnd := make([]byte, l)
  _, _ = rand.Read(rnd)
  var bld strings.Builder
  j := RandInt(0, l)
  k := (j + 1) % l
  m := (k + 2) % l
  n := (m + 3) % l
  for i, b := range rnd {
    switch i {
    case j:
      bld.WriteString(digit[int(b) % len(digit)]) // At least one digit
    case k:
      bld.WriteString(lower[int(b) % len(lower)]) // At least one lower
    case m:
      bld.WriteString(upper[int(b) % len(upper)]) // At least one upper
    case n:
      bld.WriteString(symbol[int(b) % len(symbol)]) // At least one symbol
    default:
      bld.WriteString(alnumsym[int(b) % len(alnumsym)])
    }
  }
  return bld.String()
}

func RandSymP(l int) *string {
  return stringP(RandSym(l))
}

func RandFrom[T any](items ...T) T {
  i := RandInt(0, len(items))
  return items[i]
}

func RandDate(a, b time.Time) time.Time {
  t := time.Unix(int64(RandInt(int(a.Unix()), int(b.Unix()))), 0)
  date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.UTC().Location())
  return date
}

func RandDateP(a, b time.Time) *time.Time {
  return timeP(RandDate(a, b))
}

func RandTime(a, b time.Time) time.Time {
  return time.Unix(int64(RandInt(int(a.Unix()), int(b.Unix()))), 0).UTC()
}

func RandTimeP(a, b time.Time) *time.Time {
  return timeP(RandTime(a, b))
}

func RandEmail() string {
  domain := RandFrom("mail.com", "email.com", "gmail.com")
  email := fmt.Sprintf("%s@%s", RandAbc(8), domain)
  return strings.ToLower(email)
}

func RandEmailP() *string {
  return stringP(RandEmail())
}

func RandURL() string {
  domain := RandFrom("com", "org", "net")
  url := fmt.Sprintf("https://%s.%s/%s", RandStr(8), domain, RandStr(8))
  return strings.ToLower(url)
}

func RandURLP() *string {
  return stringP(RandURL())
}

func RandIP() string {
  return fmt.Sprintf(
    "%d.%d.%d.%d",
    RandInt(0, 256), RandInt(0, 256), RandInt(0, 256), RandInt(0, 256),
  )
}

func RandIPP() *string {
  return stringP(RandIP())
}

func RandIPv6() string {
  l := 8
  groups := make([]string, l)
  for i := range l {
    groups[i] = RandHex(4)
  }
  return strings.Join(groups, ":")
}

func RandIPv6P() *string {
  return stringP(RandIPv6())
}

func RandIBAN(country string) string {
  if len(country) != 2 {
    country = "AA"
  }
  country = strings.ToUpper(country)
  code := make([]int32, len(country))
  for i, letter := range country {
    code[i] = letter - 'A' + 10
  }
  number, check := RandInt(1e11, 1e12), 0
  str := fmt.Sprintf("%012d%d%d%02d", number, code[0], code[1], check)
  var n int
  _, _ = fmt.Sscanf(str, "%d", &n)
  check = 98 - (n % 97)
  iban := fmt.Sprintf("%s%02d%012d", country, check, number)
  return iban
}

func RandIBANP(country string) *string {
  return stringP(RandIBAN(country))
}

func RandJPG() ([]byte, error) {
  width, height := 100, 100
  img := image.NewRGBA(image.Rect(0, 0, width, height))
  rnd := make([]byte, width * height * 3)
  _, _ = rand.Read(rnd)
  for y := range height {
    for x := range width {
      img.Set(x, y, color.RGBA{
        R: rnd[y * width + x * 3 + 0],
        G: rnd[y * width + x * 3 + 1],
        B: rnd[y * width + x * 3 + 2],
        A: 255,
      })
    }
  }
  var jpg bytes.Buffer
  err := jpeg.Encode(&jpg, img, &jpeg.Options{Quality: 75})
  if err != nil {
    return nil, err
  }
  return jpg.Bytes(), nil
}
