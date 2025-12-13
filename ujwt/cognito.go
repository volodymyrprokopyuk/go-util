package ujwt

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/volodymyrprokopyuk/go-util/ucheck"
	"github.com/volodymyrprokopyuk/go-util/ureq"
	"github.com/volodymyrprokopyuk/go-util/userv"
)

type jwkt struct {
  Kid string `json:"kid"`
  Kty string `json:"kty"`
  Alg string `json:"alg"`
  N string `json:"n"`
  E string `json:"e"`
}

type jwkst struct {
  Keys []*jwkt `json:"keys"`
}

type JWKS struct {
  httpc *ureq.Client
  mtx sync.RWMutex
  keys map[string]*rsa.PublicKey
}

func NewJWKS(httpc *ureq.Client) *JWKS {
  return &JWKS{
    httpc: httpc,
    keys: make(map[string]*rsa.PublicKey),
  }
}

func jwkToRSA(jwk *jwkt) (*rsa.PublicKey, error) {
  nb, err := base64.RawURLEncoding.DecodeString(jwk.N)
  if err != nil {
    return nil, err
  }
  n := new(big.Int).SetBytes(nb)
  eb, err := base64.RawURLEncoding.DecodeString(jwk.E)
  if err != nil {
    return nil, err
  }
  var e64 uint64
  switch len(eb) {
  case 3:
    e64 = uint64(binary.BigEndian.Uint32(append([]byte{0}, eb...)))
  case 4:
    e64 = uint64(binary.BigEndian.Uint32(eb))
  default:
    e64 = new(big.Int).SetBytes(eb).Uint64()
  }
  if e64 > uint64(^uint32(0)) {
    return nil, errors.New("JWK exponent too large")
  }
  e := int(e64)
  pub := &rsa.PublicKey{N: n, E: e}
  return pub, nil
}

func (c *JWKS) Fetch(ctx context.Context) error {
  var jwks jwkst
  res, err := c.httpc.GET(
    ctx, ureq.URL("/.well-known/jwks.json"), ureq.ResJSON(&jwks),
  )
  if err != nil {
    return err
  }
  if res.StatusCode != http.StatusOK {
    return fmt.Errorf(
      "JWKS fetch: expected %d, got %d", http.StatusOK, res.StatusCode,
    )
  }
  keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
  for _, jwk := range jwks.Keys {
    if jwk.Kty != "RSA" {
      continue
    }
    pub, err := jwkToRSA(jwk)
    if err != nil {
      fmt.Printf("JWK to RSA: %s\n", err)
      continue
    }
    keys[jwk.Kid] = pub
  }
  if len(keys) == 0 {
    return errors.New("JWKS fetch: empty key set")
  }
  c.mtx.Lock()
  defer c.mtx.Unlock()
  c.keys = keys
  return nil
}

func (c *JWKS) Key(kid string) (*rsa.PublicKey, bool) {
  c.mtx.RLock()
  defer c.mtx.RUnlock()
  pub, exist := c.keys[kid]
  return pub, exist
}

const (
  TokenUseAccess = "access"
  TokenUseID = "id"
)

type jwtHeader struct {
  Alg string `json:"alg"`
  Typ string `json:"typ"`
  Kid string `json:"kid"`
}

type JWTClaims struct {
  // Access token
  Iss string `json:"iss"`
  TokenUse string `json:"token_use"`
  Exp int64 `json:"exp"`
  ClientID string `json:"client_id"`
  Roles []string `json:"cognito:groups"`
  // ID token
  Aud string `json:"aud"`
  Email string `json:"email"`
}

func jwtClaimsCheck(
  claims *JWTClaims, issuer, tokenUse string,
  clientIDs []string, roles [][]string,
) error {
  // JWT issuer
  if claims.Iss != issuer {
    return userv.Unautorized("invalid JWT issuer")
  }
  // JWT use
  if claims.TokenUse != tokenUse {
    return userv.Unautorized("invalid JWT use")
  }
  // JWT expiry
  if time.Unix(claims.Exp, 0).UTC().Before(time.Now().UTC()) {
    return userv.Unautorized("expired JWT")
  }
  // JWT client ID
  switch tokenUse {
  case TokenUseAccess:
    if !slices.Contains(clientIDs, claims.ClientID) {
      return userv.Unautorized("invalid client ID")
    }
  case TokenUseID:
    if !slices.Contains(clientIDs, claims.Aud) {
      return userv.Unautorized("invalid client ID")
    }
  default:
    return userv.Unautorized("invalid token use")
  }
  // JWT roles [||] && [||]
  for _, query := range roles {
    found := ucheck.ContainsAny(claims.Roles, query)
    if found == nil {
      err := fmt.Errorf(
        "missing role: at least one of %s is required",
        strings.Join(query, ", "),
      )
      return userv.Forbidden(err.Error())
    }
  }
  return nil
}

func JWTRS256Assert(
  ctx context.Context, jwt string, jwks *JWKS, issuer, tokenUse string,
  clientIDs []string, roles [][]string, // [||] && [||]
) error {
  // Parse JWT
  parts := strings.Split(jwt, ".")
  if len(parts) != 3 {
    return userv.Unautorized("invalid JWT format")
  }
  ehead, eclaims, esig := parts[0], parts[1], parts[2]
  // Check JWT RS256 signature algorithm
  jhead, err := base64.RawURLEncoding.DecodeString(ehead)
  if err != nil {
    return userv.Unautorized("invalid JWT header encoding")
  }
  var head jwtHeader
  err = json.Unmarshal(jhead, &head)
  if err != nil {
    return userv.Unautorized("invalid JWT header format")
  }
  if head.Alg != "RS256" {
    return userv.Unautorized("unsupported JWT signature algorithm")
  }
  // Lookup verifying JWK
  pub, exist := jwks.Key(head.Kid)
  if !exist {
    // Re-fetch Cognito-rotate JWKS
    err = jwks.Fetch(ctx)
    if err != nil {
      return userv.Unautorized(err.Error())
    }
    pub, exist = jwks.Key(head.Kid)
    if !exist {
      return userv.Unautorized("JWKS kid is not found")
    }
  }
  // Verify JWT RS256 signature
  msg := fmt.Sprintf("%s.%s", ehead, eclaims)
  h := sha256.New()
  h.Write([]byte(msg))
  hash := h.Sum(nil)
  sig, err := base64.RawURLEncoding.DecodeString(esig)
  if err != nil {
    return userv.Unautorized("invalid JWT signature format")
  }
  err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, hash, sig)
  if err != nil {
    return userv.Unautorized("invalid JWT signature")
  }
  // Check JWT claims
  jclaims, err := base64.RawURLEncoding.DecodeString(eclaims)
  if err != nil {
    return userv.Unautorized("invalid JWT claims encoding")
  }
  var claims JWTClaims
  err = json.Unmarshal(jclaims, &claims)
  if err != nil {
    return userv.Unautorized("invalid JWT claims format")
  }
  return jwtClaimsCheck(&claims, issuer, tokenUse, clientIDs, roles)
}

func JWTDecodeClaims(jwt string) (*JWTClaims, error) {
  parts := strings.Split(jwt, ".")
  if len(parts) != 3 {
    return nil, errors.New("invalid JWT format")
  }
  jstr, err := base64.RawURLEncoding.DecodeString(parts[1])
  if err != nil {
    return nil, err
  }
  var claims JWTClaims
  err = json.Unmarshal([]byte(jstr), &claims)
  if err != nil {
    return nil, err
  }
  return &claims, nil
}

func JWTDecode(jwt string) (map[string]any, error) {
  parts := strings.Split(jwt, ".")
  if len(parts) != 3 {
    return nil, errors.New("invalid JWT format")
  }
  jstr, err := base64.RawURLEncoding.DecodeString(parts[1])
  if err != nil {
    return nil, err
  }
  var claims map[string]any
  err = json.Unmarshal([]byte(jstr), &claims)
  if err != nil {
    return nil, err
  }
  val, exist := claims["exp"]
  if exist {
    exp, assert := val.(float64)
    if assert {
      claims["exp"] = time.Unix(int64(exp), 0).UTC()
    }
  }
  return claims, nil
}

func MakeJWTPass() func(roles [][]string) userv.Middleware {
  return func(roles [][]string) userv.Middleware {
    return func(next http.HandlerFunc) http.HandlerFunc {
      return func(w http.ResponseWriter, r *http.Request) {
        next(w, r)
      }
    }
  }
}
