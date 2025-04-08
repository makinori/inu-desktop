package internal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

func readXauthData(buf *bytes.Buffer) ([]byte, error) {
	var length uint16
	err := binary.Read(buf, binary.BigEndian, &length)
	if err != nil {
		return []byte{}, err
	}

	data := make([]byte, length)
	_, err = io.ReadFull(buf, data)
	if err != nil {
		return []byte{}, err
	}

	return data, nil
}

type Xauth struct {
	Family  uint16
	Address string
	Number  string
	Name    string
	Data    []byte
}

func getXauthority() (Xauth, error) {
	// /usr/include/X11/Xauth.h

	filePath, envExists := os.LookupEnv("XAUTHORITY")

	if !envExists {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return Xauth{}, err
		}

		filePath = filepath.Join(homeDir, ".Xauthority")
	}

	xauthData, err := os.ReadFile(filePath)
	if err != nil {
		return Xauth{}, err
	}

	xauthBuf := bytes.NewBuffer(xauthData)

	var xauth Xauth

	binary.Read(xauthBuf, binary.BigEndian, &xauth.Family)

	address, err := readXauthData(xauthBuf)
	if err != nil {
		return xauth, err
	}
	xauth.Address = string(address)

	number, err := readXauthData(xauthBuf)
	if err != nil {
		return xauth, err
	}
	xauth.Number = string(number)

	name, err := readXauthData(xauthBuf)
	if err != nil {
		return Xauth{}, err
	}
	xauth.Name = string(name)

	xauth.Data, err = readXauthData(xauthBuf)
	if err != nil {
		return Xauth{}, err
	}

	return xauth, nil
}

func padTo4(b []byte) []byte {
	padLen := (4 - len(b)%4) % 4
	return append(b, bytes.Repeat([]byte{0x00}, padLen)...)
}

func SetupXlib() {
	xauth, err := getXauthority()
	if err != nil {
		panic(err)
	}

	conn, err := net.Dial("unix", "/tmp/.X11-unix/X0")
	if err != nil {
		log.Error("failed to connect to x11:", err)
		return
	}
	defer conn.Close()

	authNameLen := uint16(len(xauth.Name))
	authNamePadded := padTo4([]byte(xauth.Name))

	authDataLen := uint16(len(xauth.Data))
	authDataPadded := padTo4([]byte(xauth.Data))

	setup := bytes.NewBuffer([]byte{})

	binary.Write(setup, binary.LittleEndian, []byte{'l', 0}) // little endian
	binary.Write(setup, binary.LittleEndian, []byte{11, 0})  // major version
	binary.Write(setup, binary.LittleEndian, []byte{0, 0})   // minor version
	binary.Write(setup, binary.LittleEndian, authNameLen)
	binary.Write(setup, binary.LittleEndian, authDataLen)
	binary.Write(setup, binary.LittleEndian, []byte{0, 0}) // padding
	binary.Write(setup, binary.LittleEndian, authNamePadded)
	binary.Write(setup, binary.LittleEndian, authDataPadded)

	_, err = conn.Write(setup.Bytes())
	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		panic(err)
	}

	// 0 failure
	// 1 success
	// 2 auth required

	if buffer[0] != 1 {
		log.Error("failed to auth to x11")
		panic(true)
	}

	fmt.Println("Server replied:", buffer[:n])

}
