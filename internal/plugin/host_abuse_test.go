package plugin

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestWASMHostRejectsTrap(t *testing.T) {
	_, err := RunWASM(context.Background(), testHostManifest(), trapModule(), testRunInput())
	if err == nil {
		t.Fatal("RunWASM succeeded, want trap error")
	}
	if !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("RunWASM error = %v, want unreachable trap detail", err)
	}
}

func TestWASMHostRejectsInvalidUTF8Output(t *testing.T) {
	_, err := RunWASM(context.Background(), testHostManifest(), moduleWritingOutputBytes([]byte{
		'{', '"', 'a', 'c', 't', 'i', 'o', 'n', '"', ':', '"', 'r', 'e', 'p', 'l', 'a', 'c', 'e', '"',
		',', '"', 't', 'e', 'x', 't', '"', ':', '"', 0xff, '"', '}',
	}), testRunInput())
	if err == nil {
		t.Fatal("RunWASM succeeded, want invalid UTF-8 output error")
	}
	if !strings.Contains(err.Error(), "decode plugin output") {
		t.Fatalf("RunWASM error = %v, want decode detail", err)
	}
}

func TestWASMHostRejectsProtectedFieldMutationOutput(t *testing.T) {
	_, err := RunWASM(context.Background(), testHostManifest(), moduleWritingOutput(
		`{"action":"replace","text":"short","block":{"id":"other-block"}}`,
		false,
	), testRunInput())
	if err == nil {
		t.Fatal("RunWASM succeeded, want unknown field error")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("RunWASM error = %v, want unknown field detail", err)
	}
}

func TestWASMHostRejectsOversizedSelectedBlockInput(t *testing.T) {
	manifest := testHostManifest()
	manifest.Limits.MaxInputBytes = 128
	input := testRunInput()
	input.Block.Text = strings.Repeat("x", 1024)

	_, err := RunWASM(context.Background(), manifest, moduleWritingOutput(`{"action":"keep"}`, false), input)
	if err == nil {
		t.Fatal("RunWASM succeeded, want max_input_bytes error")
	}
	if !strings.Contains(err.Error(), "max_input_bytes") {
		t.Fatalf("RunWASM error = %v, want max_input_bytes detail", err)
	}
}

func trapModule() []byte {
	return buildWASM(wasmModule{
		types:     []wasmFuncType{{results: []byte{i32Type}}},
		funcTypes: []uint32{0},
		memoryMin: 1,
		memoryMax: 1,
		exports:   []wasmExport{{name: "run", kind: exportFunc, index: 0}},
		code:      [][]byte{{opUnreachable, opEnd}},
	})
}

func moduleWritingOutputBytes(output []byte) []byte {
	imports := []wasmImport{
		{module: "cachy", name: "output_write", kind: importFunc, typeIndex: 0},
	}
	types := []wasmFuncType{
		{params: []byte{i32Type, i32Type}, results: []byte{i32Type}},
		{results: []byte{i32Type}},
	}
	body := bytes.Buffer{}
	body.WriteByte(opI32Const)
	writeSLEB(&body, 1024)
	body.WriteByte(opI32Const)
	writeSLEB(&body, int32(len(output)))
	body.WriteByte(opCall)
	writeULEB(&body, 0)
	body.WriteByte(opEnd)

	return buildWASM(wasmModule{
		types:       types,
		imports:     imports,
		funcTypes:   []uint32{1},
		memoryMin:   1,
		memoryMax:   1,
		exports:     []wasmExport{{name: "memory", kind: exportMemory, index: 0}, {name: "run", kind: exportFunc, index: 1}},
		code:        [][]byte{body.Bytes()},
		dataOffset:  1024,
		dataPayload: output,
	})
}
