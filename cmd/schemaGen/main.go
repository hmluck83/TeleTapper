package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/tl"
)

func main() {
	// TL 스키마 파일 읽기
	data, err := os.ReadFile("decode/schema/telegram_api.tl")
	if err != nil {
		fmt.Printf("TL 파일 읽기 실패: %v\n", err)
		return
	}

	// TL 스키마 파싱
	schema, err := tl.Parse(bytes.NewReader(data))
	if err != nil {
		fmt.Printf("TL 파싱 실패: %v\n", err)
		return
	}

	var sb strings.Builder
	sb.WriteString("package decode\n\n")
	sb.WriteString("var ConstructorMap = map[int64]string{\n")

	count := 0
	// Constructors (types) 처리
	for _, def := range schema.Definitions {
		if def.Category == tl.CategoryType {
			fmt.Fprintf(&sb, "\t0x%08x: %q,\n", def.Definition.ID, def.Definition.Name)
			count++
		}
	}

	// Functions 처리
	for _, def := range schema.Definitions {
		if def.Category == tl.CategoryFunction {
			fmt.Fprintf(&sb, "\t0x%08x: %q,\n", def.Definition.ID, def.Definition.Name)
			count++
		}
	}

	sb.WriteString("}\n")

	if err := os.WriteFile("decode/schema.go", []byte(sb.String()), 0644); err != nil {
		fmt.Printf("파일 저장 실패: %v\n", err)
		return
	}

	fmt.Printf("decode/schema.go 파일 생성 완료 (총 %d개 definitions)\n", count)
}
