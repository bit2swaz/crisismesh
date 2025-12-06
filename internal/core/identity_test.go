package core

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/nacl/box"
)

func TestKeyGeneration(t *testing.T) {
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("GenerateIdentity failed: %v", err)
	}

	if id.NodeID == "" {
		t.Error("NodeID is empty")
	}

	pubKey, err := hex.DecodeString(id.PubKey)
	if err != nil {
		t.Fatalf("Failed to decode PubKey: %v", err)
	}
	if len(pubKey) != 32 {
		t.Errorf("Expected PubKey length 32, got %d", len(pubKey))
	}

	privKey, err := hex.DecodeString(id.PrivKey)
	if err != nil {
		t.Fatalf("Failed to decode PrivKey: %v", err)
	}
	if len(privKey) != 32 {
		t.Errorf("Expected PrivKey length 32, got %d", len(privKey))
	}
}

func TestEncryptionCycle(t *testing.T) {
	// 1. Setup Alice and Bob
	alice, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}

	// 2. Prepare Keys
	alicePriv, _ := hex.DecodeString(alice.PrivKey)
	bobPub, _ := hex.DecodeString(bob.PubKey)
	bobPriv, _ := hex.DecodeString(bob.PrivKey)

	var alicePrivKey, bobPubKey, bobPrivKey [32]byte
	copy(alicePrivKey[:], alicePriv)
	copy(bobPubKey[:], bobPub)
	copy(bobPrivKey[:], bobPriv)

	// 3. Encrypt (Alice -> Bob)
	plaintext := "Secret Plans"
	// Note: In our implementation we use SealAnonymous which uses ephemeral keys for sender anonymity
	// But here let's test the exact logic we used in gossip.go:
	// encrypted, err := box.SealAnonymous(nil, []byte(content), &pubKeyArr, rand.Reader)

	encrypted, err := box.SealAnonymous(nil, []byte(plaintext), &bobPubKey, rand.Reader)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if string(encrypted) == plaintext {
		t.Error("Ciphertext matches plaintext (encryption failed)")
	}

	// 4. Decrypt (Bob receives)
	// decrypted, ok := box.OpenAnonymous(nil, encrypted, &pubKeyArr, &privKeyArr)
	// Wait, OpenAnonymous takes (out, box, publicKey, privateKey)
	// The publicKey arg in OpenAnonymous is the *sender's* public key if using authenticated box,
	// OR the recipient's public key?
	// Let's check the docs or implementation.
	// box.SealAnonymous(out, message, recipientPubKey, rand) -> creates ephemeral sender key.
	// box.OpenAnonymous(out, box, recipientPubKey, recipientPrivKey) -> decrypts.

	decrypted, ok := box.OpenAnonymous(nil, encrypted, &bobPubKey, &bobPrivKey)
	if !ok {
		t.Fatal("Decryption failed")
	}

	if string(decrypted) != plaintext {
		t.Errorf("Decrypted text mismatch. Got %q, want %q", string(decrypted), plaintext)
	}
}

func TestDecryptionFailure(t *testing.T) {
	// Alice sends to Bob
	_, _ = GenerateIdentity() // Alice (unused)
	bob, _ := GenerateIdentity()
	charlie, _ := GenerateIdentity() // Attacker

	bobPub, _ := hex.DecodeString(bob.PubKey)
	var bobPubKey [32]byte
	copy(bobPubKey[:], bobPub)

	plaintext := "Top Secret"
	encrypted, _ := box.SealAnonymous(nil, []byte(plaintext), &bobPubKey, rand.Reader)

	// Charlie tries to decrypt
	charliePub, _ := hex.DecodeString(charlie.PubKey)
	charliePriv, _ := hex.DecodeString(charlie.PrivKey)
	var charliePubKey, charliePrivKey [32]byte
	copy(charliePubKey[:], charliePub)
	copy(charliePrivKey[:], charliePriv)

	// Charlie uses HIS keys to try and open the box sent to Bob
	_, ok := box.OpenAnonymous(nil, encrypted, &charliePubKey, &charliePrivKey)
	if ok {
		t.Error("Charlie successfully decrypted Bob's message!")
	}
}
