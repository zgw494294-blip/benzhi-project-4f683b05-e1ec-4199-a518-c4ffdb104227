package repository

import (
	"encoding/binary"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketMeta         = []byte("meta")
	bucketCases        = []byte("cases")
	bucketAccessions   = []byte("accession_codes")
	bucketStatus       = []byte("status_index")
	bucketIdempotency  = []byte("idempotency")
	bucketAudit        = []byte("audit")
	bucketCertificates = []byte("certificates")
	bucketManifests    = []byte("manifests")
)

const schemaVersion uint64 = 1

func initialize(tx *bolt.Tx) error {
	for _, name := range [][]byte{bucketMeta, bucketCases, bucketAccessions, bucketStatus, bucketIdempotency, bucketAudit, bucketCertificates, bucketManifests} {
		if _, err := tx.CreateBucketIfNotExists(name); err != nil {
			return err
		}
	}
	meta := tx.Bucket(bucketMeta)
	current := meta.Get([]byte("schemaVersion"))
	if current == nil {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, schemaVersion)
		return meta.Put([]byte("schemaVersion"), buf)
	}
	if len(current) != 8 || binary.BigEndian.Uint64(current) != schemaVersion {
		return fmt.Errorf("不兼容的schemaVersion")
	}
	return nil
}

func validateSchema(tx *bolt.Tx) error {
	meta := tx.Bucket(bucketMeta)
	if meta == nil {
		return fmt.Errorf("缺少meta数据桶")
	}
	current := meta.Get([]byte("schemaVersion"))
	if len(current) != 8 || binary.BigEndian.Uint64(current) != schemaVersion {
		return fmt.Errorf("不兼容的schemaVersion")
	}
	for _, name := range [][]byte{bucketCases, bucketAccessions, bucketStatus, bucketIdempotency, bucketAudit, bucketCertificates, bucketManifests} {
		if tx.Bucket(name) == nil {
			return fmt.Errorf("缺少数据桶%s", name)
		}
	}
	return nil
}

func statusKey(status, id string) []byte { return []byte(status + "\x00" + id) }
func auditKey(caseID string, sequence uint64) []byte {
	buf := make([]byte, len(caseID)+1+8)
	copy(buf, caseID)
	buf[len(caseID)] = 0
	binary.BigEndian.PutUint64(buf[len(caseID)+1:], sequence)
	return buf
}

func certificateKey(serial uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, serial)
	return buf
}
