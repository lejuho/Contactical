package types

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"fmt"
)

// VerifyDataSignature verifies that the data was signed by the registered TEE key
// pubKeyBytes: DER-encoded public key from NodeInfo
// signature: ECDSA signature over the signed data
// sensorHash, gnssHash: original data hashes
// timestamp: Unix timestamp when data was signed
func VerifyDataSignature(pubKeyBytes []byte, signature []byte, sensorHash, gnssHash string, timestamp int64) error {
	// 1. Parse public key
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// 2. Assert key type (Android TEE typically uses ECDSA P-256)
	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("unsupported key type: expected ECDSA, got %T", pubKey)
	}

	// 3. Reconstruct the signed data
	// The data format must match what Android app signs
	signedData := BuildSignedData(sensorHash, gnssHash, timestamp)

	// 4. Hash the data (SHA-256)
	hash := sha256.Sum256(signedData)

	// 5. Verify the signature
	if !ecdsa.VerifyASN1(ecdsaPubKey, hash[:], signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// BuildSignedData constructs the data that should be signed
// Format: sensor_hash || gnss_hash || timestamp (8 bytes big-endian)
// This must match the format used by the Android app
func BuildSignedData(sensorHash, gnssHash string, timestamp int64) []byte {
	// sensor_hash + gnss_hash as bytes
	data := []byte(sensorHash + gnssHash)

	// Append timestamp as 8 bytes big-endian
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(timestamp))
	data = append(data, timestampBytes...)

	return data
}

// VerifyDataSignatureRSA verifies signature using RSA key (fallback for older devices)
func VerifyDataSignatureRSA(pubKeyBytes []byte, signature []byte, sensorHash, gnssHash string, timestamp int64) error {
	// Parse public key
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPubKey, ok := pubKey.(crypto.PublicKey)
	if !ok {
		return fmt.Errorf("unsupported key type")
	}

	// Reconstruct the signed data
	signedData := BuildSignedData(sensorHash, gnssHash, timestamp)

	// Hash the data
	hash := sha256.Sum256(signedData)

	// For RSA, we would use rsa.VerifyPKCS1v15 or rsa.VerifyPSS
	// But Android TEE typically uses ECDSA, so this is a fallback
	_ = rsaPubKey
	_ = hash

	return fmt.Errorf("RSA verification not implemented - use ECDSA keys")
}

// VerifyDataSignatureAuto automatically detects key type and verifies
func VerifyDataSignatureAuto(pubKeyBytes []byte, signature []byte, sensorHash, gnssHash string, timestamp int64) error {
	// Parse public key to detect type
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	switch key := pubKey.(type) {
	case *ecdsa.PublicKey:
		return VerifyDataSignature(pubKeyBytes, signature, sensorHash, gnssHash, timestamp)
	default:
		return fmt.Errorf("unsupported key type: %T", key)
	}
}
