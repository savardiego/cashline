package keys

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/base58"
	bip39 "github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/ripemd160"
)

// DiceSeqRequiredLength is the number of required dice results
const DiceSeqRequiredLength = 99

// CoinflipSeqRequiredLength is the number of required coin flip results
const CoinflipSeqRequiredLength = 256

// HexSeqRequiredLength is the number of required hex chars
const HexSeqRequiredLength = 64

var maxValueForKey *big.Int
var minValueForKey *big.Int

func init() {
	maxValueForKey = new(big.Int)
	maxValueForKey.SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364140", 16)
	minValueForKey = new(big.Int)
	minValueForKey.SetString("1", 16)
}

// PrivateFromWIF decodes a base58 encoded key (compressed or uncompressed) (WIF Wallet Import Format) to []byte
func PrivateFromWIF(keyString string) (key []byte, compressed bool, err error) {
	// Decoding key using base58
	decoded := base58.Decode(keyString)
	if decoded[0] != 0x80 {
		return nil, false, fmt.Errorf("input value is not a valid mainnet key")
	}
	checkSum := decoded[len(decoded)-4:]
	hashOne := sha256.Sum256(decoded[:len(decoded)-4])
	hashTwo := sha256.Sum256(hashOne[:])
	newCheckSum := hashTwo[:4]
	if string(newCheckSum) != string(checkSum) {
		return nil, false, fmt.Errorf("cannot decode private key %v because checksum is wrong", key)
	}
	decKey := decoded[1 : len(decoded)-4]
	key = decKey
	compressed = false
	if len(decKey) == 33 && decKey[len(decKey)-1] == 0x01 {
		compressed = true
		key = decKey[:32]
	}
	return key, compressed, nil
}

// FromDiceSequence returns a private key generated from a base6 sequence of 99 0-5 chars
func FromDiceSequence(sequence string) (key []byte, err error) {
	if len(sequence) != DiceSeqRequiredLength {
		return nil, fmt.Errorf("given sequence is %d long, must be %d", len(sequence), DiceSeqRequiredLength)
	}
	privKey, err := diceKey(sequence)
	if err != nil {
		return nil, fmt.Errorf("cannot read sequence: %v", err)
	}
	return privKey, nil
}

// FromCoinflipSequence returns a private key generated from a base2 sequence of 256 0-1 chars
func FromCoinflipSequence(sequence string) (key []byte, err error) {
	if len(sequence) != CoinflipSeqRequiredLength {
		return nil, fmt.Errorf("given sequence is %d long, must be %d", len(sequence), CoinflipSeqRequiredLength)
	}
	privKey, err := coinflipsKey(sequence)
	if err != nil {
		return nil, fmt.Errorf("cannot read sequence: %v", err)
	}
	return privKey, nil
}

// ToWIF encode a private key (given as a hex string) to WIF (Wallet IMport Format) compressed or uncompressed
func ToWIF(privKey []byte, compressed bool) (string, error) {
	first := append([]byte{0x80}, privKey...)
	if compressed {
		first = append(first, 0x01)
	}
	second := sha256.Sum256(first)
	third := sha256.Sum256(second[:])
	checksum := third[:4]
	fourth := append(first, checksum...)
	encoded := base58.Encode(fourth)
	return encoded, nil
}

// Public derivates a public key in compressed or uncompressed format from a private key
func Public(privateKey []byte, compressed bool) (pubKey []byte) {
	publicKey := derivatePublicKey(privateKey)
	if compressed {
		pubKey = toCompressedBytes(publicKey)
	} else {
		pubKey = toUncompressedBytes(publicKey)
	}
	return pubKey
}

// Hashed returns the hashed (sha256 + ripemd160) version of the public key
func Hashed(pubKey []byte) []byte {
	sha256Hash := sha256.Sum256(pubKey)
	ripe160 := ripemd160.New()
	ripe160.Write(sha256Hash[:])
	hash := ripe160.Sum(nil)
	return hash
}

func derivatePublicKey(key []byte) ecdsa.PublicKey {
	bigNumberKey := new(big.Int)
	bigNumberKey.SetBytes(key)
	privKey := new(ecdsa.PrivateKey)
	privKey.D = bigNumberKey
	secp256k1Curve := btcec.S256()
	privKey.PublicKey.Curve = secp256k1Curve
	privKey.PublicKey.X, privKey.PublicKey.Y = secp256k1Curve.ScalarBaseMult(bigNumberKey.Bytes())
	publicKey := privKey.PublicKey
	return publicKey
}

func toCompressedBytes(pubK ecdsa.PublicKey) (compressedPubKey []byte) {
	byteX := pubK.X.Bytes()
	//byteY := pubK.Y.Bytes()
	yIsEven := isEven(pubK.Y) //O means X is even, 1 means X is odd
	compressedPubKey = []byte{}
	//Append 0x02 if Y even and 0x03 if X is odd
	if yIsEven {
		compressedPubKey = append(compressedPubKey, 0x02)
	} else {
		compressedPubKey = append(compressedPubKey, 0x03)
	}
	compressedPubKey = append(compressedPubKey, byteX...)
	return compressedPubKey
}

func toUncompressedBytes(pubK ecdsa.PublicKey) (uncompressedPubKey []byte) {
	byteX := pubK.X.Bytes()
	byteY := pubK.Y.Bytes()
	//Append 0x04 X and Y to build public key
	uncompressedPubKey = []byte{0x04}
	uncompressedPubKey = append(uncompressedPubKey, byteX...)
	uncompressedPubKey = append(uncompressedPubKey, byteY...)
	return uncompressedPubKey
}

func isValidKey(keyNum *big.Int) bool {
	var notTooSmall bool
	var notTooBig bool
	if keyNum.Cmp(minValueForKey) >= 0 {
		notTooSmall = true
	}
	if keyNum.Cmp(maxValueForKey) <= 0 {
		notTooBig = true
	}
	if notTooSmall && notTooBig {
		return true
	}
	return false
}

func isEven(num *big.Int) (even bool) {
	evenOdd := num.Bit(0) //O means X is even, 1 means X is odd
	// defer func() { fmt.Printf("Number %v is even: %v\n", num, even) }()
	even = true
	if evenOdd == 1 {
		even = false
	}
	return
}

// Mnemonic generates a mnemonic from a byte array
func Mnemonic(seed []byte) (string, error) {
	mnemonic, err := bip39.NewMnemonic(seed)
	if err != nil {
		return "", fmt.Errorf("cennot generate mnemonic: %v", err)
	}
	return mnemonic, nil
}

func coinflipsKey(sequence string) ([]byte, error) {
	bi := new(big.Int)
	bi, ok := bi.SetString(sequence, 2)
	if !ok {
		return nil, fmt.Errorf("big.Int.SetString return false for sequence %v", sequence)
	}
	if !isValidKey(bi) {
		return nil, errors.New("input sequence represents a number not acceptable as private key")
	}
	return bi.Bytes(), nil
}

func diceKey(sequence string) ([]byte, error) {
	basesix := ""
	for _, c := range []byte(sequence) {
		n, err := strconv.ParseInt(string(c), 10, 8)
		if err != nil {
			return nil, fmt.Errorf("problems with char %c due to: %v", c, err)
		}
		basesix += strconv.Itoa(int(n - 1))
	}
	bi := new(big.Int)
	bi, ok := bi.SetString(basesix, 6)
	if !ok {
		return nil, fmt.Errorf("big.Int.SetString return false for sequence %v", basesix)
	}
	if !isValidKey(bi) {
		return nil, errors.New("input sequence represents a number not acceptable as private key")
	}
	return bi.Bytes(), nil
}
