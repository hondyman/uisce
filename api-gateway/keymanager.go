package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"github.com/golang-jwt/jwt/v5"
)

type KeyManager struct {
	mu      sync.RWMutex
	keys    map[string]*rsa.PrivateKey
	current string
}

func NewKeyManager() *KeyManager {
	km := &KeyManager{keys: make(map[string]*rsa.PrivateKey)}
	// If persistence configured, attempt to load keys from disk
	persistPath := os.Getenv("KEYMANAGER_PERSIST_PATH")
	pass := os.Getenv("KEYMANAGER_PASSPHRASE")
	if persistPath != "" && pass != "" {
		if err := km.LoadFromFile(persistPath, pass); err != nil {
			// If load fails, still create a new key and continue
			fmt.Printf("warning: failed to load persisted keys: %v\n", err)
			km.RotateKey()
			// Attempt to persist newly generated key
			_ = km.SaveToFile(persistPath, pass)
		}
		return km
	}

	// Default behavior: generate an initial key
	km.RotateKey()
	return km
}

// RotateKey generates a new RSA key and makes it the current signing key.
func (km *KeyManager) RotateKey() (string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	kid := fmt.Sprintf("kid-%d", time.Now().UnixNano())
	km.mu.Lock()
	km.keys[kid] = priv
	km.current = kid
	km.mu.Unlock()
	// Persist to disk if configured
	if path := os.Getenv("KEYMANAGER_PERSIST_PATH"); path != "" {
		pass := os.Getenv("KEYMANAGER_PASSPHRASE")
		if pass != "" {
			if err := km.SaveToFile(path, pass); err != nil {
				fmt.Printf("warning: failed to persist keys after rotation: %v\n", err)
			}
		}
	}
	return kid, nil
}

func (km *KeyManager) GetCurrent() (string, *rsa.PrivateKey) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.current, km.keys[km.current]
}

func (km *KeyManager) GetPublicKey(kid string) (*rsa.PublicKey, bool) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	if k, ok := km.keys[kid]; ok {
		return &k.PublicKey, true
	}
	return nil, false
}

// JWKS returns the public keys in JWKS format
func (km *KeyManager) JWKSHandler(w http.ResponseWriter, r *http.Request) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	keys := make([]map[string]interface{}, 0, len(km.keys))
	for kid, pk := range km.keys {
		n := base64.RawURLEncoding.EncodeToString(pk.N.Bytes())
		eBytes := big.NewInt(int64(pk.E)).Bytes()
		e := base64.RawURLEncoding.EncodeToString(eBytes)
		jwk := map[string]interface{}{
			"kty": "RSA",
			"kid": kid,
			"use": "sig",
			"alg": "RS256",
			"n":   n,
			"e":   e,
		}
		keys = append(keys, jwk)
	}

	out := map[string]interface{}{"keys": keys}
	b, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// RotateKeyHandler rotates the key and returns the new kid
func (km *KeyManager) RotateKeyHandler(w http.ResponseWriter, r *http.Request) {
	kid, err := km.RotateKey()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("rotate error: %v", err)))
		return
	}
	// Optionally persist keys to disk if KEYMANAGER_PERSIST_PATH is set (simple demo)
	if path := os.Getenv("KEYMANAGER_PERSIST_PATH"); path != "" {
		// naive persistence: write serialized current kid only (for demo)
		_ = os.WriteFile(path, []byte(kid), 0600)
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, fmt.Sprintf(`{"kid":"%s"}`, kid))
}

// SignTokenRS256 signs claims with current RSA key and returns a signed token string and kid
func (km *KeyManager) SignTokenRS256(claims jwt.Claims) (string, string, error) {
	kid, priv := km.GetCurrent()
	if priv == nil {
		return "", "", fmt.Errorf("no signing key available")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(priv)
	if err != nil {
		return "", "", err
	}
	return signed, kid, nil
}

// SaveToFile writes all current private keys (PEM) encrypted to the given path using passphrase.
func (km *KeyManager) SaveToFile(path, passphrase string) error {
	km.mu.RLock()
	defer km.mu.RUnlock()

	type stored struct {
		Nonce string `json:"nonce"`
		Data  string `json:"data"`
		Salt  string `json:"salt"`
		Iter  int    `json:"iter"`
	}
	out := struct {
		Keys    map[string]stored `json:"keys"`
		Current string            `json:"current"`
	}{Keys: make(map[string]stored), Current: km.current}

	for kid, pk := range km.keys {
		der := x509.MarshalPKCS1PrivateKey(pk)
		pemBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
		pemBytes := pem.EncodeToMemory(pemBlock)

		// Use PBKDF2-derived key per-key with a random salt so the file is resistant to precomputation
		iter := 100000
		if v := os.Getenv("KEYMANAGER_PBKDF2_ITER"); v != "" {
			if iv, err := strconv.Atoi(v); err == nil && iv > 0 {
				iter = iv
			}
		}
		nonce, ct, salt, err := encryptWithPBKDF2(pemBytes, passphrase, iter)
		if err != nil {
			return err
		}
		out.Keys[kid] = stored{Nonce: base64.RawURLEncoding.EncodeToString(nonce), Data: base64.RawURLEncoding.EncodeToString(ct), Salt: base64.RawURLEncoding.EncodeToString(salt), Iter: iter}
	}

	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

// LoadFromFile loads and decrypts persisted keys from path using passphrase.
func (km *KeyManager) LoadFromFile(path, passphrase string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	type stored struct {
		Nonce string `json:"nonce"`
		Data  string `json:"data"`
		Salt  string `json:"salt"`
		Iter  int    `json:"iter"`
	}
	in := struct {
		Keys    map[string]stored `json:"keys"`
		Current string            `json:"current"`
	}{}
	if err := json.Unmarshal(b, &in); err != nil {
		return err
	}
	km.mu.Lock()
	defer km.mu.Unlock()
	for kid, s := range in.Keys {
		nonce, err := base64.RawURLEncoding.DecodeString(s.Nonce)
		if err != nil {
			return err
		}
		ct, err := base64.RawURLEncoding.DecodeString(s.Data)
		if err != nil {
			return err
		}
		salt, err := base64.RawURLEncoding.DecodeString(s.Salt)
		if err != nil {
			return err
		}
		iter := s.Iter
		if iter == 0 {
			iter = 100000
		}
		plain, err := decryptWithPBKDF2(nonce, ct, passphrase, salt, iter)
		if err != nil {
			return err
		}
		// parse PEM
		blk, _ := pem.Decode(plain)
		if blk == nil {
			return fmt.Errorf("invalid PEM for kid %s", kid)
		}
		pk, err := x509.ParsePKCS1PrivateKey(blk.Bytes)
		if err != nil {
			return err
		}
		km.keys[kid] = pk
	}
	km.current = in.Current
	return nil
}

// encryptWithPBKDF2 derives a key from passphrase and a random salt and encrypts using AES-GCM.
func encryptWithPBKDF2(plain []byte, passphrase string, iter int) (nonce, ct, salt []byte, err error) {
	salt = make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return nil, nil, nil, err
	}
	key := pbkdf2.Key([]byte(passphrase), salt, iter, 32, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, nil, err
	}
	ct = gcm.Seal(nil, nonce, plain, nil)
	return nonce, ct, salt, nil
}

// decryptWithPBKDF2 derives key from passphrase and salt and decrypts ciphertext using AES-GCM.
func decryptWithPBKDF2(nonce, ct []byte, passphrase string, salt []byte, iter int) ([]byte, error) {
	key := pbkdf2.Key([]byte(passphrase), salt, iter, 32, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}
