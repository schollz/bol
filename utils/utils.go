package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"os"
	"strings"
	"time"
)

func HashAndHex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

// GetRandomMD5Hash returns 8 bytes of a random md5 hash
func GetRandomMD5Hash() string {
	hasher := md5.New()
	hasher.Write([]byte(RandStringBytesMaskImprSrc(10, time.Now().UnixNano())))
	return hex.EncodeToString(hasher.Sum(nil))[0:8]
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandStringBytesMaskImprSrc generates a random string using a alphabet and seed
// from SO
func RandStringBytesMaskImprSrc(n int, seed int64) string {
	src := rand.NewSource(seed)
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// StrExtract extracts a string between two other strings
// from http://stackoverflow.com/questions/21000277/extract-text-content-from-html-in-golang
func StrExtract(sExper, sAdelim, sCdelim string, nOccur int) string {
	aExper := strings.Split(sExper, sAdelim)
	if len(aExper) <= nOccur {
		return ""
	}
	sMember := aExper[nOccur]
	aExper = strings.Split(sMember, sCdelim)
	if len(aExper) == 1 {
		return ""
	}
	return aExper[0]
}

// Exists returns whether the given file or directory exists or not
// from http://stackoverflow.com/questions/10510691/how-to-check-whether-a-file-or-directory-denoted-by-a-path-exists-in-golang
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// Shred writes random data to the file before erasing it
func Shred(fileName string) error {
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	fileData, err := f.Stat()
	if err != nil {
		return err
	}
	b := make([]byte, fileData.Size())
	_, err = rand.Read(b)
	if err != nil {
		return err
	}
	_, err = f.WriteAt(b, 0)
	if err != nil {
		return err
	}
	f.Close()
	err = os.Remove(fileName)
	if err != nil {
		return err
	}
	return nil
}
