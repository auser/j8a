package j8a

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/hako/durafmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"math"
	"math/big"
	"strings"
	"time"
)

type PDuration time.Duration

func (p PDuration) AsString() string {
	return durafmt.Parse(time.Duration(p)).LimitFirstN(2).String()
}

func (p PDuration) AsDuration() time.Duration {
	return time.Duration(p)
}

func (p PDuration) AsDays() int {
	return int(p.AsDuration().Hours() / 24)
}

type TlsLink struct {
	cert              *x509.Certificate
	issued            time.Time
	remainingValidity PDuration
	totalValidity     PDuration
	browserValidity   PDuration
	earliestExpiry    bool
	isCA              bool
}

func (t TlsLink) browserExpiry() PDuration {
	return PDuration(time.Hour * 24 * 398)
}

func (t TlsLink) printRemainingValidity() string {
	rv := t.remainingValidity.AsString()
	if t.earliestExpiry {
		rv = rv + ", which is the earliest in your chain"
	}
	return rv
}

func (r *Runtime) tlsHealthCheck(daemon bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Trace().Msgf("TLS cert not analysed, cause: %s", r)
		}
	}()

	//safety first
	if r.ReloadableCert.Cert != nil {
	Daemon:
		for {
			//Andeka is checking our certificate chains forever.
			andeka, _ := checkFullCertChain(r.ReloadableCert.Cert)
			logCertStats(andeka)
			if daemon {
				time.Sleep(time.Hour * 24)
			} else {
				break Daemon
			}
		}
	}
}

func checkFullCertChainFromBytes(cert []byte, key []byte) ([]TlsLink, error) {
	var chain tls.Certificate

	var e1 error
	chain, e1 = tls.X509KeyPair(cert, key)
	if e1 != nil {
		return nil, e1
	}
	return checkFullCertChain(&chain)
}

func checkFullCertChain(chain *tls.Certificate) ([]TlsLink, error) {
	if len(chain.Certificate) == 0 {
		return nil, errors.New("no certificate data found")
	}

	var e2 error
	chain.Leaf, e2 = x509.ParseCertificate(chain.Certificate[0])
	if e2 != nil {
		return nil, e2
	}

	if chain.Leaf.DNSNames == nil || len(chain.Leaf.DNSNames) == 0 {
		return nil, errors.New("no DNS name specified")
	}

	inter, root, e3 := splitCertPools(chain)
	if e3 != nil {
		return nil, e3
	}

	verified, e4 := chain.Leaf.Verify(verifyOptions(inter, root))
	if e4 != nil {
		return nil, e4
	}

	return parseTlsLinks(verified[0]), nil
}

func verifyOptions(inter *x509.CertPool, root *x509.CertPool) x509.VerifyOptions {
	opts := x509.VerifyOptions{}
	if inter != nil && len(inter.Subjects()) > 0 {
		opts.Intermediates = inter
	}
	if root != nil && len(root.Subjects()) > 0 {
		opts.Roots = root
	}
	return opts
}

func formatSerial(serial *big.Int) string {
	serial = serial.Abs(serial)
	hex := fmt.Sprintf("%X", serial)
	if len(hex)%2 != 0 {
		hex = "0" + hex
	}
	if len(hex) > 2 {
		frm := strings.Builder{}
		for i := 0; i < len(hex); i += 2 {
			var j = 0
			if i+2 <= len(hex) {
				j = i + 2
			} else {
				j = len(hex)
			}
			w := hex[i:j]
			frm.WriteString(w)
			if i < len(hex)-2 {
				frm.WriteString(":")
			}
		}
		hex = frm.String()
	}
	return hex
}

func sha1Fingerprint(cert *x509.Certificate) string {
	sha1 := sha1.Sum(cert.Raw)
	return "#" + JoinHashString(sha1[:])
}

func sha256Fingerprint(cert *x509.Certificate) string {
	sha256 := sha256.Sum256(cert.Raw)
	return "#" + JoinHashString(sha256[:])
}

func md5Fingerprint(cert *x509.Certificate) string {
	md5 := md5.Sum(cert.Raw)
	return "#" + JoinHashString(md5[:])
}

func JoinHashString(hash []byte) string {
	return strings.Join(ChunkString(strings.ToUpper(hex.EncodeToString(hash[:])), 2), ":")
}

func ChunkString(s string, chunkSize int) []string {
	var chunks []string
	runes := []rune(s)

	if len(runes) == 0 {
		return []string{s}
	}

	for i := 0; i < len(runes); i += chunkSize {
		nn := i + chunkSize
		if nn > len(runes) {
			nn = len(runes)
		}
		chunks = append(chunks, string(runes[i:nn]))
	}
	return chunks
}

func logCertStats(tlsLinks []TlsLink) {
	month := PDuration(time.Hour * 24 * 30)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Snapshot of your cert chain size %d explained. ", len(tlsLinks)))
	for i, link := range tlsLinks {
		if !link.isCA {
			sb.WriteString(fmt.Sprintf("[%d/%d] TLS cert serial #%s, sha1 fingerprint %s for DNS names %s, valid from %s, signed by [%s], expires in %s. ",
				i+1,
				len(tlsLinks),
				formatSerial(link.cert.SerialNumber),
				sha1Fingerprint(link.cert),
				link.cert.DNSNames,
				link.issued.Format("2006-01-02"),
				link.cert.Issuer.CommonName,
				link.printRemainingValidity(),
			))
			if link.totalValidity > link.browserExpiry() {
				sb.WriteString(fmt.Sprintf("Total validity period of %d days is above legal browser maximum of %d days. ",
					int(link.totalValidity.AsDays()),
					int(link.browserExpiry().AsDays())))
			}
			if link.browserValidity < 0 {
				sb.WriteString(fmt.Sprintf("Validity grace period expired %s ago, update this certificate now to avoid disruption. ",
					link.browserValidity.AsString()))
			} else if link.browserValidity < link.remainingValidity {
				sb.WriteString(fmt.Sprintf("You may experience disruption in %s. ",
					link.browserValidity.AsString()))
			}
		} else {
			caType := "Intermediate"
			if isRoot(link.cert) {
				caType = "Root"
			}
			sb.WriteString(fmt.Sprintf("[%d/%d] %s CA #%s Common name [%s], signed by [%s], expires in %s. ",
				i+1,
				len(tlsLinks),
				caType,
				formatSerial(link.cert.SerialNumber),
				link.cert.Subject.CommonName,
				link.cert.Issuer.CommonName,
				link.remainingValidity.AsString(),
			))
		}
	}

	for _, t := range tlsLinks {
		if t.earliestExpiry {
			var ev *zerolog.Event
			//if the certificate expires in less than 30 days we send this as a log.Warn event instead.
			if t.remainingValidity < month {
				ev = log.Warn()
			} else {
				ev = log.Debug()
			}
			ev.Msg(sb.String())
		}
	}
}

func parseTlsLinks(chain []*x509.Certificate) []TlsLink {
	earliestExpiry := PDuration(math.MaxInt64)
	browserExpiry := TlsLink{}.browserExpiry().AsDuration()
	var tlsLinks []TlsLink
	si := 0
	for i, cert := range chain {
		link := TlsLink{
			cert:              cert,
			issued:            cert.NotBefore,
			remainingValidity: PDuration(time.Until(cert.NotAfter)),
			totalValidity:     PDuration(cert.NotAfter.Sub(cert.NotBefore)),
			browserValidity:   PDuration(time.Until(cert.NotBefore.Add(browserExpiry))),
			earliestExpiry:    false,
			isCA:              cert.IsCA,
		}
		tlsLinks = append(tlsLinks, link)
		if link.remainingValidity < earliestExpiry {
			si = i
			earliestExpiry = link.remainingValidity
		}
	}
	tlsLinks[si].earliestExpiry = true
	return tlsLinks
}

func splitCertPools(chain *tls.Certificate) (*x509.CertPool, *x509.CertPool, error) {
	var err error

	root := x509.NewCertPool()
	inter := x509.NewCertPool()
	for _, c := range chain.Certificate {
		c1, caerr := x509.ParseCertificate(c)
		if caerr != nil {
			err = caerr
		}
		//for CA's we treat you as intermediate unless you signed yourself
		if c1.IsCA {
			//as above, you're intermediate in the last position unless you signed yourself, that makes you a root cert.
			if isRoot(c1) {
				root.AddCert(c1)
			} else {
				inter.AddCert(c1)
			}
		}
	}
	return inter, root, err
}

func isRoot(c *x509.Certificate) bool {
	return c.IsCA && c.Issuer.CommonName == c.Subject.CommonName
}
