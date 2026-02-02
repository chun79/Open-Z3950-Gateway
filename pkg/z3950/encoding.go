package z3950

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// DecodeText 智能尝试将字节流转换为 UTF-8 字符串
func DecodeText(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	if utf8.Valid(data) {
		return string(data)
	}

	// 优先尝试 CJK 常见编码
	// 很多时候探测器会误判 GBK 为 Latin1 (windows-1252)，所以优先尝试 CJK
	encoders := []encoding.Encoding{
		simplifiedchinese.GBK,
		traditionalchinese.Big5,
		japanese.ShiftJIS,
		japanese.EUCJP,
		korean.EUCKR,
	}

	for _, enc := range encoders {
		decoded, err := doDecode(data, enc)
		if err == nil && !strings.Contains(decoded, "\uFFFD") {
			return decoded
		}
	}

	// 尝试探测
	e, _, _ := charset.DetermineEncoding(data, "")
	if e != nil {
		decoded, err := doDecode(data, e)
		if err == nil && !strings.Contains(decoded, "\uFFFD") {
			return decoded
		}
	}

	return string(data)
}

func doDecode(data []byte, enc encoding.Encoding) (string, error) {
	reader := transform.NewReader(bytes.NewReader(data), enc.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}