package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/frida/frida-go/frida"
	"github.com/gotd/td/bin"
	"github.com/hmluck83/TeleTapper/decode"
)

// Frida에서 받는 메시지 구조체
type FridaMessage struct {
	Type    string         `json:"type"`
	Payload MessagePayload `json:"payload"`
}

type MessagePayload struct {
	Direction string `json:"direction"`
	Data      []byte `json:"data"`
}

// MTProto 메시지 헤더 정보
type MTProtoHeader struct {
	Salt          int64
	SessionID     int64
	MessageID     int64
	SeqNo         uint32
	MessageLength uint32
}

// handle MTProto Message
func processMTProtoMessage(direction string, data []byte) {
	if len(data) < 32 {
		fmt.Println("Buffer too small to parse MTProto message")
		return
	}

	// MTProto 헤더 파싱
	header := MTProtoHeader{
		Salt:          int64(binary.LittleEndian.Uint64(data[0:8])),
		SessionID:     int64(binary.LittleEndian.Uint64(data[8:16])),
		MessageID:     int64(binary.LittleEndian.Uint64(data[16:24])),
		SeqNo:         binary.LittleEndian.Uint32(data[24:28]),
		MessageLength: binary.LittleEndian.Uint32(data[28:32]),
	}

	dirLabel := "📤 SEND"
	if direction == "receive" {
		dirLabel = "📥 RECV"
	}

	fmt.Printf("%s  ", dirLabel)
	// 메시지 본문 추출 및 디코딩
	if len(data) >= 32+int(header.MessageLength) {
		messageBody := data[32 : 32+header.MessageLength]
		decode.HandleMessage(header.MessageID, &bin.Buffer{Buf: messageBody}, direction)
	}
}

func loadHookScript() (string, error) {
	// 실행 파일 경로 기준으로 스크립트 경로 찾기
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// 프로젝트 루트에서 scripts/hook.js 찾기
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(execPath)))
	scriptPath := filepath.Join(projectRoot, "scripts", "hook.js")

	// 파일이 없으면 현재 작업 디렉토리 기준으로 시도
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		scriptPath = filepath.Join(cwd, "scripts", "hook.js")
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read hook script from %s: %w", scriptPath, err)
	}

	return string(content), nil
}

func main() {
	// Hook 스크립트 로드
	hookScript, err := loadHookScript()
	if err != nil {
		fmt.Printf("Failed to load hook script: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[*] Hook script loaded from file")

	// Device Manager 생성
	deviceManager := frida.NewDeviceManager()
	localDevice, err := deviceManager.LocalDevice()
	if err != nil {
		fmt.Printf("Failed to get local device: %v\n", err)
		os.Exit(1)
	}

	// Telegram 실행 파일 경로
	telegramPath := "Telegram/Telegram"
	fmt.Printf("[*] Spawning %s...\n", telegramPath)

	// Telegram 프로세스 spawn
	opts := frida.NewSpawnOptions()
	opts.SetStdio(frida.StdioPipe)

	pid, err := localDevice.Spawn(telegramPath, opts)
	if err != nil {
		fmt.Printf("Failed to spawn process: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[*] Spawned Telegram with PID: %d\n", pid)

	// Spawn된 프로세스에 attach
	session, err := localDevice.Attach(pid, nil)
	if err != nil {
		fmt.Printf("Failed to attach to process: %v\n", err)
		os.Exit(1)
	}
	defer session.Detach()

	fmt.Println("[*] Loading hook script...")

	// 스크립트 생성 및 로드
	script, err := session.CreateScript(hookScript)
	if err != nil {
		fmt.Printf("Failed to create script: %v\n", err)
		os.Exit(1)
	}

	// 메시지 핸들러 설정
	script.On("message", func(message string) {
		var fridaMsg FridaMessage
		err := json.Unmarshal([]byte(message), &fridaMsg)
		if err != nil {
			// JSON 파싱 실패 시 원본 메시지 출력
			fmt.Println("🚫 Error on parsing Message")
			return
		}

		processMTProtoMessage(fridaMsg.Payload.Direction, fridaMsg.Payload.Data)
	})

	if err := script.Load(); err != nil {
		fmt.Printf("Failed to load script: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[*] Hook loaded successfully.")

	// 프로세스 재개
	err = localDevice.Resume(pid)
	if err != nil {
		fmt.Printf("Failed to resume process: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[*] Telegram resumed. Monitoring messages...")
	fmt.Println("[*] Press Ctrl+C to stop")

	// 시그널 대기
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n[*] Stopping...")
	script.Unload()
}
