package sourceconn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConnConfig_FromParts(t *testing.T) {
	cc, host, db, ssl, warns, err := connConfig([]byte(`{"host":"src.example","port":5433,"database":"sales","username":"ro","password":"pw","schema":"crm","sslMode":"require"}`))
	require.NoError(t, err)
	require.Empty(t, warns)
	require.Equal(t, "src.example", host)
	require.Equal(t, "sales", db)
	require.Equal(t, "require", ssl)
	require.EqualValues(t, 5433, cc.Port)
	require.Equal(t, "ro", cc.User)
	require.Equal(t, "pw", cc.Password)
	require.Equal(t, "crm", cc.RuntimeParams["search_path"])
}

func TestConnConfig_NestedAuthWinsAndSSLDefaults(t *testing.T) {
	cc, _, _, ssl, _, err := connConfig([]byte(`{"host":"h","port":5432,"database":"d","username":"flat","password":"flatpw","auth":{"basic":{"username":"nested","password":"nestedpw"}}}`))
	require.NoError(t, err)
	require.Equal(t, "nested", cc.User)
	require.Equal(t, "nestedpw", cc.Password)
	require.Equal(t, "disable", ssl, "no ssl flag and no sslMode is disable, as before")

	_, _, _, ssl, _, err = connConfig([]byte(`{"host":"h","port":5432,"database":"d","username":"u","ssl":true}`))
	require.NoError(t, err)
	require.Equal(t, "require", ssl)
}

func TestConnConfig_DSNPassThrough(t *testing.T) {
	cc, host, db, _, _, err := connConfig([]byte(`{"dsn":"postgres://u:p@dsnhost:6543/dsndb?sslmode=disable"}`))
	require.NoError(t, err)
	require.Equal(t, "dsnhost", host)
	require.Equal(t, "dsndb", db)
	require.EqualValues(t, 6543, cc.Port)
}

func TestConnConfig_RefusesIncompleteOrMalformed(t *testing.T) {
	for name, raw := range map[string]string{
		"not json":     `nope`,
		"empty host":   `{"port":5432,"database":"d","username":"u"}`,
		"port zero":    `{"host":"h","port":0,"database":"d","username":"u"}`,
		"port too big": `{"host":"h","port":70000,"database":"d","username":"u"}`,
		"no username":  `{"host":"h","port":5432,"database":"d"}`,
		"no database":  `{"host":"h","port":5432,"username":"u"}`,
		"unparseable":  `{"dsn":"postgres://u@h:notaport/d"}`,
		"empty object": `{}`,
		"json null":    `null`,
	} {
		_, _, _, _, _, err := connConfig([]byte(raw))
		require.ErrorIs(t, err, ErrBadConfig, name)
	}
}

func selfSigned(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "client"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	require.NoError(t, err)
	kb, err := x509.MarshalECPrivateKey(k)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}))
}

func TestConnConfig_MutualTLS(t *testing.T) {
	cert, key := selfSigned(t)
	base := `"host":"h","port":5432,"database":"d","username":"u","auth_type":"key_pair"`
	esc := func(s string) string {
		b, _ := jsonString(s)
		return b
	}

	t.Run("a CA verifies the server", func(t *testing.T) {
		cc, _, _, _, warns, err := connConfig([]byte(`{` + base + `,"client_cert":` + esc(cert) + `,"private_key":` + esc(key) + `,"ca_cert":` + esc(cert) + `}`))
		require.NoError(t, err)
		require.Empty(t, warns)
		require.NotNil(t, cc.TLSConfig)
		require.False(t, cc.TLSConfig.InsecureSkipVerify)
		require.Len(t, cc.TLSConfig.Certificates, 1)
	})
	t.Run("no CA means the server is not verified, and that is reported", func(t *testing.T) {
		cc, _, _, _, warns, err := connConfig([]byte(`{` + base + `,"client_cert":` + esc(cert) + `,"private_key":` + esc(key) + `}`))
		require.NoError(t, err)
		require.True(t, cc.TLSConfig.InsecureSkipVerify)
		require.Len(t, warns, 1)
		require.Contains(t, warns[0], "will not be verified")
	})
	t.Run("a bad key pair or CA is refused", func(t *testing.T) {
		_, _, _, _, _, err := connConfig([]byte(`{` + base + `,"client_cert":` + esc(cert) + `,"private_key":"junk"}`))
		require.ErrorIs(t, err, ErrBadConfig)
		_, _, _, _, _, err = connConfig([]byte(`{` + base + `,"client_cert":` + esc(cert) + `,"private_key":` + esc(key) + `,"ca_cert":"junk"}`))
		require.ErrorIs(t, err, ErrBadConfig)
	})
}
