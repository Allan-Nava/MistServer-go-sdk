package lib

import (
	"crypto/md5"
	"encoding/hex"
)

func GenerateMD5(input string) string {
	hash := md5.Sum([]byte(input))     // Compute MD5 hash
	return hex.EncodeToString(hash[:]) // Convert to hexadecimal string
}
