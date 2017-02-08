package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/schollz/cryptopasta"
)

func GetPassword() string {
	fmt.Printf("Enter password: ")
	var password string
	if runtime.GOOS == "windows" {
		fmt.Scanln(&password) // not great fix, but works for cygwin
	} else {
		bytePassword, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
		password = strings.TrimSpace(string(bytePassword))
	}
	return password
}

func EncryptToFile(toEncrypt []byte, password string, filename string) error {
	key := sha256.Sum256([]byte(password))
	encrypted, err := cryptopasta.Encrypt(toEncrypt, &key)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, []byte(hex.EncodeToString(encrypted)), 0755)
}

func DecryptFromFile(password string, filename string) ([]byte, error) {
	key := sha256.Sum256([]byte(password))
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return []byte{}, err
	}
	contentData, err := hex.DecodeString(string(content))
	if err != nil {
		return []byte{}, err
	}
	return cryptopasta.Decrypt(contentData, &key)
}

func HashAndHex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

// GetRandomMD5Hash returns 8 bytes of a random md5 hash
func GetRandomMD5Hash() string {
	hasher := md5.New()
	hasher.Write([]byte(RandStringBytesMaskImprSrc(10)))
	return hex.EncodeToString(hasher.Sum(nil))[0:8]
}

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// stole this from stack overflow.
func RandStringBytesMaskImprSrc(n int) string {
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
	if fileData.IsDir() {
		// it's a directory
		return errors.New("Can't shred directory")
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

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
//http://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

var DATE_FORMAT = "2006-01-02 15:04:05"

func GetCurrentDate() string {
	// git date format: Thu, 07 Apr 2005 22:13:13 +0200
	formattedTime := time.Now().Format(DATE_FORMAT)
	return formattedTime
}

func ParseDate(date string) (time.Time, error) {
	date = strings.TrimSpace(date)
	newTime, err := time.Parse(time.RFC1123Z, date)
	if err != nil {
		newTime, err = time.Parse(time.RFC3339, date)
	}
	if err != nil {
		newTime, err = time.Parse("2006-01-02 15:04:05", date)
	}
	if err != nil {
		newTime, err = time.Parse("Mon Jan 02 15:04:05 2006", date)
	}
	if err != nil {
		newTime, err = time.Parse("Mon Jan 02 15:04:05 2006 -0700", date)
	}
	if err != nil {
		newTime, err = time.Parse("Mon Jan 2 15:04:05 2006 -0700", date)
	}
	if err != nil {
		newTime, err = time.Parse("Mon, Jan 02 15:04:05 2006 -0700", date)
	}
	if err != nil {
		newTime, err = time.Parse("Mon 02 Jan 2006 15:04:05 -0700", date)
	}
	if err != nil {
		newTime, err = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", date)
	}
	if err != nil {
		newTime, err = time.Parse("2006-01-02 15:04", date)
	}
	if err != nil {
		newTime, err = time.Parse("2006-01-02", date)
	}
	if err != nil {
		newTime, err = time.Parse("2006-01-02T15:04:05-07:00", date)
	}
	return newTime, err
}

func FormatDate(date time.Time) string {
	return date.Format(DATE_FORMAT)
}

func ReFormatDate(date string) string {
	parsedDate, _ := ParseDate(date)
	return FormatDate(parsedDate)
}

func GetUnixTimestamp() string {
	return strconv.Itoa(int(time.Now().UnixNano() / 1000000))
}
