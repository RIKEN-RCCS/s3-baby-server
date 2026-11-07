// Copyright (c) 2026 Quantova Inc
// SPDX-License-Identifier: BSD-2-Clause

package attestation_test

import (
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"

	"s3-baby-server/attestation"
)

func manifestPayload() []byte {
	return attestation.ObjectManifest{
		Key:    "bucket/object-001.dat",
		Size:   1048576,
		SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}.Canonical()
}

func mustKeypair(t *testing.T) (publicKey, secretKey []byte) {
	t.Helper()

	publicKey, secretKey, err := attestation.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}

	return publicKey, secretKey
}

func TestBuildAndVerifyRoundtrip(t *testing.T) {
	payload := manifestPayload()
	publicKey, secretKey := mustKeypair(t)

	a, err := attestation.Build(payload, publicKey, secretKey)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if a.Scheme != attestation.Scheme || a.Issuer != attestation.Issuer {
		t.Fatalf("header mismatch: %+v", a)
	}

	if !attestation.Verify(payload, a, nil) {
		t.Fatal("Verify returned false for a valid attestation")
	}

	if !attestation.Verify(payload, a, publicKey) {
		t.Fatal("Verify with a pinned public key returned false")
	}
}

func TestTamperedObjectIsRejected(t *testing.T) {
	payload := manifestPayload()
	publicKey, secretKey := mustKeypair(t)

	a, err := attestation.Build(payload, publicKey, secretKey)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	altered := attestation.ObjectManifest{
		Key:    "bucket/object-001.dat",
		Size:   1048576,
		SHA256: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}.Canonical()
	if attestation.Verify(altered, a, nil) {
		t.Fatal("Verify accepted a tampered object manifest")
	}
}

func TestWrongSignerIsRejectedWhenPinned(t *testing.T) {
	payload := manifestPayload()
	publicKey, secretKey := mustKeypair(t)
	otherPublicKey, _ := mustKeypair(t)

	a, err := attestation.Build(payload, publicKey, secretKey)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if attestation.Verify(payload, a, otherPublicKey) {
		t.Fatal("Verify accepted an attestation under a different pinned key")
	}
}

func TestForgedSignatureIsRejected(t *testing.T) {
	payload := manifestPayload()
	publicKey, secretKey := mustKeypair(t)

	a, err := attestation.Build(payload, publicKey, secretKey)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	a.Signature = base64.StdEncoding.EncodeToString(make([]byte, mldsa65.SignatureSize))
	if attestation.Verify(payload, a, nil) {
		t.Fatal("Verify accepted a forged signature")
	}
}

func TestVerifyRejectsMalformedFields(t *testing.T) {
	payload := manifestPayload()
	publicKey, secretKey := mustKeypair(t)

	a, err := attestation.Build(payload, publicKey, secretKey)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	bad := *a
	bad.PublicKey = "!!!not base64!!!"

	if attestation.Verify(payload, &bad, nil) {
		t.Fatal("Verify accepted a non base64 public key")
	}

	bad = *a
	bad.PublicKey = base64.StdEncoding.EncodeToString([]byte("short"))

	if attestation.Verify(payload, &bad, nil) {
		t.Fatal("Verify accepted a public key of the wrong length")
	}
}

func TestBuildRejectsBadSecretKey(t *testing.T) {
	payload := manifestPayload()
	publicKey, _ := mustKeypair(t)

	if _, err := attestation.Build(payload, publicKey, []byte("short")); err == nil {
		t.Fatal("Build accepted a malformed secret key")
	}
}

func TestSignerFileRoundtrip(t *testing.T) {
	payload := manifestPayload()
	publicKey, secretKey := mustKeypair(t)
	path := filepath.Join(t.TempDir(), "signer.json")

	if err := attestation.SaveSigner(path, publicKey, secretKey); err != nil {
		t.Fatalf("SaveSigner: %v", err)
	}

	loadedPublic, loadedSecret, err := attestation.LoadSigner(path)
	if err != nil {
		t.Fatalf("LoadSigner: %v", err)
	}

	a, err := attestation.Build(payload, loadedPublic, loadedSecret)
	if err != nil {
		t.Fatalf("Build with loaded signer: %v", err)
	}

	if !attestation.Verify(payload, a, publicKey) {
		t.Fatal("attestation from a loaded signer did not verify")
	}
}

func TestManifestCanonicalIsStable(t *testing.T) {
	m := attestation.ObjectManifest{Key: "k", Size: 10, SHA256: "ab"}
	if string(m.Canonical()) != string(m.Canonical()) {
		t.Fatal("Canonical encoding is not stable")
	}
}
