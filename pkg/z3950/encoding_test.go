package z3950

import (
	"bytes"
	"io"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func TestDecodeText(t *testing.T) {
	// 1. UTF-8
	utf8Str := "Hello, 世界"
	if got := DecodeText([]byte(utf8Str)); got != utf8Str {
		t.Errorf("UTF-8 decode failed. Got %q, want %q", got, utf8Str)
	}

	// 2. GBK
	// "你好，世界" might be too short for detection or have overlapping byte sequences. 
	// Use a longer, more distinct sentence: "这是一个测试句子，用于验证GBK编码的自动识别功能。"
	gbkStr := "这是一个测试句子，用于验证GBK编码的自动识别功能。"
	gbkBytes, _ := encodeToGBK(gbkStr)
	
	// Explicitly log bytes for debugging if needed
	// t.Logf("GBK Bytes: %x", gbkBytes)
	
	got := DecodeText(gbkBytes)
	if got != gbkStr {
		t.Errorf("GBK decode failed.\nGot:  %q\nWant: %q\nBytes: %x", got, gbkStr, gbkBytes)
	}

	// 3. ASCII / Empty
	if got := DecodeText([]byte("")); got != "" {
		t.Errorf("Empty decode failed. Got %q", got)
	}
	if got := DecodeText([]byte("abc")); got != "abc" {
		t.Errorf("ASCII decode failed. Got %q", got)
	}
}

func encodeToGBK(s string) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(s)), simplifiedchinese.GBK.NewEncoder())
	return io.ReadAll(reader)
}
