package certstore

import (
	"fmt"
	"strings"
	"sync"

	"github.com/off-sync/platform-proxy/app/interfaces"
	"github.com/off-sync/platform-proxy/domain/certs"
	"github.com/off-sync/platform-proxy/infra/filesystem"
)

// FileSystemCertStore implements filesystem based storage for certificates.
type FileSystemCertStore struct {
	sync.Mutex
	fs filesystem.FileSystem
}

// NewFileSystemCertStore creates a new filesystem-backed certificate store.
func NewFileSystemCertStore(fs filesystem.FileSystem) *FileSystemCertStore {
	return &FileSystemCertStore{
		fs: fs,
	}
}

const (
	certSuffix = "-crt.pem"
	keySuffix  = "-key.pem"
)

func getDomainsPath(domains []string) string {
	escaped := make([]string, len(domains))

	for i, domain := range domains {
		escaped[i] = getDomainPath(domain)
	}

	return strings.Join(escaped, "+")
}

func getDomainPath(domain string) string {
	parts := strings.Split(domain, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return strings.Join(parts, "_")
}

// LoadOrGenerate tries to retrieve a certificate for a domain.
// It tries to generate one if not found and a generator is provided.
func (s *FileSystemCertStore) LoadOrGenerate(domains []string, gen interfaces.CertGen) (*certs.Certificate, error) {
	s.Lock()
	defer s.Unlock()

	path := getDomainsPath(domains)

	certPath := path + certSuffix
	exists, err := s.fs.FileExists(certPath)
	if err != nil {
		return nil, err
	}

	if !exists {
		if gen == nil {
			return nil, nil
		}

		return gen.GenCert(domains)
	}

	certBytes, err := s.fs.ReadBytes(certPath)
	if err != nil {
		return nil, fmt.Errorf("reading certificate from path '%s': %s", certPath, err)
	}

	keyPath := path + keySuffix
	if exists, err := s.fs.FileExists(keyPath); !exists || err != nil {
		return nil, err
	}

	keyBytes, err := s.fs.ReadBytes(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key from path '%s': %s", keyPath, err)
	}

	return &certs.Certificate{
		Certificate: certBytes,
		PrivateKey:  keyBytes,
	}, nil
}

// Save stores a certificate for a domain for future retrieval.
func (s *FileSystemCertStore) Save(domains []string, crt *certs.Certificate) error {
	s.Lock()
	defer s.Unlock()

	path := getDomainsPath(domains)

	certPath := path + certSuffix
	if err := s.fs.WriteBytes(certPath, crt.Certificate); err != nil {
		return fmt.Errorf("writing certificate to path '%s': %s", certPath, err)
	}

	keyPath := path + keySuffix
	if err := s.fs.WriteBytes(keyPath, crt.PrivateKey); err != nil {
		return fmt.Errorf("writing private key to path '%s': %s", keyPath, err)
	}

	return nil
}
