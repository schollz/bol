package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/schollz/bol/utils"
)

var decryptFile, encryptFile string

func init() {
	flag.StringVar(&decryptFile, "decrypt", "", "`file` to decrypt`")
	flag.StringVar(&encryptFile, "encrypt", "", "`file` to encrypt`")
}

func main() {
	flag.Parse()
	fileName := ""
	if len(decryptFile) > 0 {
		fileName = decryptFile
	} else if len(encryptFile) > 0 {
		fileName = encryptFile
	}
	if len(fileName) > 0 {
		if utils.Exists(fileName) {
			password := utils.GetPassword()
			if len(encryptFile) > 0 {
				b, _ := ioutil.ReadFile(fileName)
				utils.EncryptToFile(b, password, fileName+".bol")
				fmt.Printf("Encrypted as %s", fileName+".bol")
			} else {
				b, err := utils.DecryptFromFile(password, fileName)
				if err == nil {
					fmt.Println(string(b))
				} else {
					fmt.Println("Incorrect password")
				}
			}
		} else {
			fmt.Printf("%s does not exist", fileName)
		}
	} else {
		fmt.Println("The boltool is compatible with the bol style encryption/decryption.")
		fmt.Println("boltool --decrypt FILE\nboltool --encrypt FILE")
	}
}
