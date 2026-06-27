package plugin

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestWASMHostRunsWithSelectedBlockEnvelope(t *testing.T) {
	result, err := RunWASM(context.Background(), testHostManifest(), moduleWritingOutput(
		`{"action":"replace","text":"short","summary":"compressed","confidence":1,"lossiness":"lossy"}`,
		true,
	), RunInput{
		RequestID:   "req_123",
		Provider:    "openai",
		Model:       "gpt-test",
		ContentType: "log",
		Block: BlockInput{
			ID:   "block_1",
			Role: "tool",
			Text: "full selected tool output",
		},
		Metadata: map[string]any{"source": "tool_result"},
	})
	if err != nil {
		t.Fatalf("RunWASM returned error: %v", err)
	}

	if result.Action != "replace" || result.Text != "short" {
		t.Fatalf("RunWASM result = %#v, want replace short", result)
	}
}

func TestWASMHostRejectsWASIImportsByDefault(t *testing.T) {
	_, err := RunWASM(context.Background(), testHostManifest(), moduleImporting("wasi_snapshot_preview1", "fd_write"), testRunInput())
	if err == nil {
		t.Fatal("RunWASM succeeded, want import error")
	}
	if !strings.Contains(err.Error(), "wasi_snapshot_preview1") {
		t.Fatalf("RunWASM error = %v, want WASI import detail", err)
	}
}

func TestWASMHostRejectsUnexpectedHostImports(t *testing.T) {
	_, err := RunWASM(context.Background(), testHostManifest(), moduleImporting("env", "tcp_connect"), testRunInput())
	if err == nil {
		t.Fatal("RunWASM succeeded, want import error")
	}
	if !strings.Contains(err.Error(), "module[env] not instantiated") {
		t.Fatalf("RunWASM error = %v, want missing host module detail", err)
	}
}

func TestWASMHostEnforcesTimeout(t *testing.T) {
	manifest := testHostManifest()
	manifest.Limits.TimeoutMS = 1

	_, err := RunWASM(context.Background(), manifest, infiniteLoopModule(), testRunInput())
	if err == nil {
		t.Fatal("RunWASM succeeded, want timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("RunWASM error = %v, want deadline exceeded", err)
	}
}

func TestWASMHostEnforcesMemoryLimit(t *testing.T) {
	manifest := testHostManifest()
	manifest.Limits.MemoryMB = 1

	_, err := RunWASM(context.Background(), manifest, moduleWithMemoryPages(20), testRunInput())
	if err == nil {
		t.Fatal("RunWASM succeeded, want memory limit error")
	}
	if !strings.Contains(err.Error(), "memory") {
		t.Fatalf("RunWASM error = %v, want memory detail", err)
	}
}

func TestWASMHostRejectsOversizedOutput(t *testing.T) {
	manifest := testHostManifest()
	manifest.Limits.MaxOutputBytes = 64

	_, err := RunWASM(context.Background(), manifest, moduleWritingOutput(
		`{"action":"replace","text":"`+strings.Repeat("x", 128)+`"}`,
		false,
	), testRunInput())
	if err == nil {
		t.Fatal("RunWASM succeeded, want output limit error")
	}
	if !strings.Contains(err.Error(), "max_output_bytes") {
		t.Fatalf("RunWASM error = %v, want max_output_bytes detail", err)
	}
}

func TestWASMHostRejectsInvalidOutput(t *testing.T) {
	_, err := RunWASM(context.Background(), testHostManifest(), moduleWritingOutput("not-json", false), testRunInput())
	if err == nil {
		t.Fatal("RunWASM succeeded, want invalid output error")
	}
	if !strings.Contains(err.Error(), "decode plugin output") {
		t.Fatalf("RunWASM error = %v, want decode detail", err)
	}
}

func testHostManifest() Manifest {
	manifest := validManifest()
	manifest.Limits.TimeoutMS = 100
	manifest.Limits.MemoryMB = 4
	manifest.Limits.MaxInputBytes = 4096
	manifest.Limits.MaxOutputBytes = 4096
	return manifest
}

func testRunInput() RunInput {
	return RunInput{
		RequestID:   "req_123",
		Provider:    "openai",
		Model:       "gpt-test",
		ContentType: "log",
		Block: BlockInput{
			ID:   "block_1",
			Role: "tool",
			Text: "selected block only",
		},
	}
}

func moduleWritingOutput(output string, readInput bool) []byte {
	imports := []wasmImport{
		{module: "cachy", name: "input_len", kind: importFunc, typeIndex: 0},
		{module: "cachy", name: "input_read", kind: importFunc, typeIndex: 1},
		{module: "cachy", name: "output_write", kind: importFunc, typeIndex: 2},
	}
	types := []wasmFuncType{
		{results: []byte{i32Type}},
		{params: []byte{i32Type}, results: []byte{i32Type}},
		{params: []byte{i32Type, i32Type}, results: []byte{i32Type}},
		{results: []byte{i32Type}},
	}
	body := bytes.Buffer{}
	if readInput {
		body.WriteByte(opCall)
		writeULEB(&body, 0)
		body.WriteByte(opDrop)
		body.WriteByte(opI32Const)
		writeSLEB(&body, 512)
		body.WriteByte(opCall)
		writeULEB(&body, 1)
		body.WriteByte(opDrop)
	}
	body.WriteByte(opI32Const)
	writeSLEB(&body, 1024)
	body.WriteByte(opI32Const)
	writeSLEB(&body, int32(len(output)))
	body.WriteByte(opCall)
	writeULEB(&body, 2)
	body.WriteByte(opEnd)

	return buildWASM(wasmModule{
		types:       types,
		imports:     imports,
		funcTypes:   []uint32{3},
		memoryMin:   1,
		memoryMax:   1,
		exports:     []wasmExport{{name: "memory", kind: exportMemory, index: 0}, {name: "run", kind: exportFunc, index: 3}},
		code:        [][]byte{body.Bytes()},
		dataOffset:  1024,
		dataPayload: []byte(output),
	})
}

func moduleImporting(module, name string) []byte {
	return buildWASM(wasmModule{
		types:     []wasmFuncType{{}},
		imports:   []wasmImport{{module: module, name: name, kind: importFunc, typeIndex: 0}},
		funcTypes: []uint32{0},
		memoryMin: 1,
		memoryMax: 1,
		exports:   []wasmExport{{name: "run", kind: exportFunc, index: 1}},
		code:      [][]byte{{opEnd}},
	})
}

func infiniteLoopModule() []byte {
	return buildWASM(wasmModule{
		types:     []wasmFuncType{{results: []byte{i32Type}}},
		funcTypes: []uint32{0},
		memoryMin: 1,
		memoryMax: 1,
		exports:   []wasmExport{{name: "run", kind: exportFunc, index: 0}},
		code:      [][]byte{{opLoop, 0x40, opBr, 0, opEnd, opI32Const, 0, opEnd}},
	})
}

func moduleWithMemoryPages(pages uint32) []byte {
	return buildWASM(wasmModule{
		types:     []wasmFuncType{{results: []byte{i32Type}}},
		funcTypes: []uint32{0},
		memoryMin: pages,
		memoryMax: pages,
		exports:   []wasmExport{{name: "memory", kind: exportMemory, index: 0}, {name: "run", kind: exportFunc, index: 0}},
		code:      [][]byte{{opI32Const, 0, opEnd}},
	})
}

const (
	i32Type = 0x7f

	importFunc   = 0x00
	exportFunc   = 0x00
	exportMemory = 0x02

	opLoop        = 0x03
	opUnreachable = 0x00
	opBr          = 0x0c
	opCall        = 0x10
	opReturn      = 0x0f
	opDrop        = 0x1a
	opI32Const    = 0x41
	opEnd         = 0x0b
)

type wasmModule struct {
	types       []wasmFuncType
	imports     []wasmImport
	funcTypes   []uint32
	memoryMin   uint32
	memoryMax   uint32
	exports     []wasmExport
	code        [][]byte
	dataOffset  uint32
	dataPayload []byte
}

type wasmFuncType struct {
	params  []byte
	results []byte
}

type wasmImport struct {
	module    string
	name      string
	kind      byte
	typeIndex uint32
}

type wasmExport struct {
	name  string
	kind  byte
	index uint32
}

func buildWASM(module wasmModule) []byte {
	var wasm bytes.Buffer
	wasm.Write([]byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00})
	writeSection(&wasm, 1, encodeTypeSection(module.types))
	if len(module.imports) > 0 {
		writeSection(&wasm, 2, encodeImportSection(module.imports))
	}
	if len(module.funcTypes) > 0 {
		writeSection(&wasm, 3, encodeFunctionSection(module.funcTypes))
	}
	if module.memoryMin > 0 {
		writeSection(&wasm, 5, encodeMemorySection(module.memoryMin, module.memoryMax))
	}
	if len(module.exports) > 0 {
		writeSection(&wasm, 7, encodeExportSection(module.exports))
	}
	if len(module.code) > 0 {
		writeSection(&wasm, 10, encodeCodeSection(module.code))
	}
	if len(module.dataPayload) > 0 {
		writeSection(&wasm, 11, encodeDataSection(module.dataOffset, module.dataPayload))
	}
	return wasm.Bytes()
}

func encodeTypeSection(types []wasmFuncType) []byte {
	var section bytes.Buffer
	writeULEB(&section, uint32(len(types)))
	for _, typ := range types {
		section.WriteByte(0x60)
		writeByteVec(&section, typ.params)
		writeByteVec(&section, typ.results)
	}
	return section.Bytes()
}

func encodeImportSection(imports []wasmImport) []byte {
	var section bytes.Buffer
	writeULEB(&section, uint32(len(imports)))
	for _, imp := range imports {
		writeName(&section, imp.module)
		writeName(&section, imp.name)
		section.WriteByte(imp.kind)
		writeULEB(&section, imp.typeIndex)
	}
	return section.Bytes()
}

func encodeFunctionSection(funcTypes []uint32) []byte {
	var section bytes.Buffer
	writeULEB(&section, uint32(len(funcTypes)))
	for _, typeIndex := range funcTypes {
		writeULEB(&section, typeIndex)
	}
	return section.Bytes()
}

func encodeMemorySection(min, max uint32) []byte {
	var section bytes.Buffer
	writeULEB(&section, 1)
	section.WriteByte(0x01)
	writeULEB(&section, min)
	writeULEB(&section, max)
	return section.Bytes()
}

func encodeExportSection(exports []wasmExport) []byte {
	var section bytes.Buffer
	writeULEB(&section, uint32(len(exports)))
	for _, export := range exports {
		writeName(&section, export.name)
		section.WriteByte(export.kind)
		writeULEB(&section, export.index)
	}
	return section.Bytes()
}

func encodeCodeSection(functions [][]byte) []byte {
	var section bytes.Buffer
	writeULEB(&section, uint32(len(functions)))
	for _, body := range functions {
		var function bytes.Buffer
		writeULEB(&function, 0)
		function.Write(body)
		writeULEB(&section, uint32(function.Len()))
		section.Write(function.Bytes())
	}
	return section.Bytes()
}

func encodeDataSection(offset uint32, payload []byte) []byte {
	var section bytes.Buffer
	writeULEB(&section, 1)
	section.WriteByte(0x00)
	section.WriteByte(opI32Const)
	writeSLEB(&section, int32(offset))
	section.WriteByte(opEnd)
	writeByteVec(&section, payload)
	return section.Bytes()
}

func writeSection(wasm *bytes.Buffer, id byte, payload []byte) {
	wasm.WriteByte(id)
	writeULEB(wasm, uint32(len(payload)))
	wasm.Write(payload)
}

func writeByteVec(w *bytes.Buffer, bytes []byte) {
	writeULEB(w, uint32(len(bytes)))
	w.Write(bytes)
}

func writeName(w *bytes.Buffer, name string) {
	writeByteVec(w, []byte(name))
}

func writeULEB(w *bytes.Buffer, value uint32) {
	for {
		b := byte(value & 0x7f)
		value >>= 7
		if value != 0 {
			b |= 0x80
		}
		w.WriteByte(b)
		if value == 0 {
			return
		}
	}
}

func writeSLEB(w *bytes.Buffer, value int32) {
	for {
		b := byte(value & 0x7f)
		value >>= 7
		done := (value == 0 && b&0x40 == 0) || (value == -1 && b&0x40 != 0)
		if !done {
			b |= 0x80
		}
		w.WriteByte(b)
		if done {
			return
		}
	}
}
