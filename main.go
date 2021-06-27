package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var programScanner *bufio.Scanner
var data []byte
var dataPtr uint16
var stdinReader *bufio.Reader
var program *instruction
var labelCount = 0

type instructionType int

const (
	nop instructionType = iota
	move
	add
	print
	read
	loop
	clear
)

type instruction struct {
	instructionType instructionType
	value           uint16
	offset          uint16
	next            *instruction
	loop            *instruction
}

func parseProgram() *instruction {
	firstInstruction := &instruction{instructionType: nop}
	currentInstruction := firstInstruction
	for currentInstruction != nil && programScanner.Scan() {
		nextInstruction := &instruction{offset: 0, value: 1}
		switch programScanner.Text() {
		case "<":
			nextInstruction.value = math.MaxUint16 // -1
			fallthrough
		case ">":
			nextInstruction.instructionType = move
		case "-":
			nextInstruction.value = math.MaxUint16 // -1
			fallthrough
		case "+":
			nextInstruction.instructionType = add
		case ".":
			nextInstruction.instructionType = print
		case ",":
			nextInstruction.instructionType = read
		case "[":
			nextInstruction.instructionType = loop
			nextInstruction.loop = parseProgram()
		case "]":
			nextInstruction = nil
		}
		currentInstruction.next = nextInstruction
		currentInstruction = nextInstruction
	}
	return firstInstruction
}

func runInstruction(instruction *instruction) {
	for instruction != nil {
		switch instruction.instructionType {
		case nop:
		case move:
			dataPtr += instruction.value
		case add:
			data[dataPtr+instruction.offset] += byte(instruction.value)
		case print:
			fmt.Printf("%c", data[dataPtr+instruction.offset])
		case read:
			data[dataPtr+instruction.offset], _ = stdinReader.ReadByte()
		case loop:
			for data[dataPtr] != 0 {
				runInstruction(instruction.loop)
			}
		case clear:
			data[dataPtr+instruction.offset] = 0
		}
		instruction = instruction.next
	}
}

func compileProgram(instruction *instruction) string {
	compiledProgram := `%define SYS_READ 0
%define SYS_WRITE 1
%define SYS_EXIT 60

%define STDIN 0
%define STDOUT 1

section .bss
data:
	resb 65536

section .text
global _start
_start:
	mov rbx, data
	mov r8, 0

`
	compiledProgram += compileInstruction(instruction)
	compiledProgram += `
    mov rax, SYS_EXIT
    mov rdi, 0
    syscall`
	return compiledProgram
}

func compileInstruction(instruction *instruction) string {
	compiledProgram := ""
	for instruction != nil {
		switch instruction.instructionType {
		case nop:
		case move:
			if instruction.value == 1 {
				compiledProgram += fmt.Sprintf("	inc r8w\n")
			} else if instruction.value == math.MaxUint16 { // -1
				compiledProgram += fmt.Sprintf("	dec r8w\n")
			} else {
				compiledProgram += fmt.Sprintf("	add r8w, %d\n", instruction.value)
			}
		case add:
			if instruction.value == 1 {
				compiledProgram += fmt.Sprintf("	inc byte [rbx+r8+%d]\n", instruction.offset)
			} else if instruction.value == math.MaxUint16 { // -1
				compiledProgram += fmt.Sprintf("	dec byte [rbx+r8+%d]\n", instruction.offset)
			} else {
				compiledProgram += fmt.Sprintf("	add [rbx+r8+%d], byte %d\n", instruction.offset, byte(instruction.value))
			}

		case print:
			compiledProgram += fmt.Sprintf(`	mov rax, SYS_WRITE
	lea rsi, [rbx+r8+%d]
	mov rdi, STDOUT
	mov rdx, 1
	syscall
`, instruction.offset)
		case read:
			compiledProgram += fmt.Sprintf(`	mov rax, SYS_READ
	lea rsi, [rbx+r8+%d]
	mov rdi, STDIN
	mov edx, 1
	syscall
`, instruction.offset)
		case loop:
			beginningLabelNum := labelCount
			labelCount++
			loopProgramm := compileInstruction(instruction.loop)
			endLabelNum := labelCount
			labelCount++
			compiledProgram += "	cmp [rbx+r8], byte 0\n"
			compiledProgram += fmt.Sprintf("	je label%d\n", endLabelNum)
			compiledProgram += fmt.Sprintf("label%d:\n", beginningLabelNum)
			compiledProgram += loopProgramm
			compiledProgram += "	cmp [rbx+r8], byte 0\n"
			compiledProgram += fmt.Sprintf("	jne label%d\n", beginningLabelNum)
			compiledProgram += fmt.Sprintf("label%d:\n", endLabelNum)
		case clear:
			compiledProgram += fmt.Sprintf("	mov [rbx+r8+%d], byte 0\n", byte(instruction.offset))
		default:
			panic("unreachable")
		}
		instruction = instruction.next
	}
	return compiledProgram
}

func removeNops(instruction *instruction) *instruction {
	if instruction == nil {
		return nil
	}
	switch instruction.instructionType {
	case nop:
		return removeNops(instruction.next)
	case loop:
		instruction.loop = removeNops(instruction.loop)
		fallthrough
	default:
		instruction.next = removeNops(instruction.next)
		return instruction
	}
}

func optimizeAdjacentInstructions(instruction *instruction) {
	if instruction == nil {
		return
	}
	switch instruction.instructionType {
	case move, add:
		for instruction.next != nil && instruction.next.instructionType == instruction.instructionType && instruction.next.offset == instruction.offset {
			instruction.value += instruction.next.value
			instruction.next = instruction.next.next
		}
	case loop:
		optimizeAdjacentInstructions(instruction.loop)
	}
	optimizeAdjacentInstructions(instruction.next)
}

func optimizeClearLoops(instruction *instruction) {
	if instruction == nil {
		return
	}
	optimizeClearLoops(instruction.next)
	if instruction.instructionType == loop {
		if instruction.loop != nil && instruction.loop.instructionType == add && instruction.loop.next == nil {
			instruction.instructionType = clear
			instruction.loop = nil
		} else {
			optimizeClearLoops(instruction.loop)
		}
	}
}

func optimizeOffsets(inst *instruction) {
	if inst == nil {
		return
	}
	currentInstruction := inst
	var offset uint16 = 0
	for currentInstruction != nil && currentInstruction.instructionType != loop {
		if currentInstruction.instructionType == move {
			offset += currentInstruction.value
			currentInstruction.instructionType = nop
		} else {
			currentInstruction.offset = offset
		}
		if currentInstruction.next == nil || currentInstruction.next.instructionType == loop {
			break
		}
		currentInstruction = currentInstruction.next
	}
	if offset != 0 {
		currentInstruction.next = &instruction{instructionType: move, value: offset, next: currentInstruction.next}
		currentInstruction = currentInstruction.next
	}
	if currentInstruction != nil {
		optimizeOffsets(currentInstruction.next)
		if currentInstruction.instructionType == loop {
			optimizeOffsets(currentInstruction.loop)
		}
	}
}

func optimizeProgram(offsets bool) {
	program = removeNops(program)
	optimizeAdjacentInstructions(program)
	optimizeClearLoops(program)
	if offsets {
		optimizeOffsets(program)
		program = removeNops(program) // remove nops generated by optimizeOffsets
	}
}

func main() {
	args := os.Args
	if len(args) < 2 || (args[1] == "--compile" && len(args) < 3) {
		fmt.Printf("Usage: %s [--compile] file\n", args[0])
		os.Exit(0)
	}
	compile := false
	file := args[1]

	if args[1] == "--compile" {
		compile = true
		file = args[2]
	}

	programFile, err := os.Open(file)
	checkError(err)
	programScanner = bufio.NewScanner(programFile)
	programScanner.Split(bufio.ScanRunes)
	program = parseProgram()

	optimizeProgram(!compile)

	if !compile {
		data = make([]byte, math.MaxUint16+1)
		stdinReader = bufio.NewReader(os.Stdin)
		runInstruction(program)
		return
	}

	compiledProgram := compileProgram(program)
	baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	tmpFolder, err := os.MkdirTemp("", "brainfuck-*")
	checkError(err)
	defer os.RemoveAll(tmpFolder)
	tmpFolder += "/"
	assemblyFileName := tmpFolder + baseName + ".asm"
	outputFile, err := os.Create(assemblyFileName)
	checkError(err)
	_, err = outputFile.WriteString(compiledProgram)
	checkError(err)
	objectFileName := tmpFolder + baseName + ".o"
	err = exec.Command("nasm", "-f", "elf64", "-o", objectFileName, assemblyFileName).Run()
	checkError(err, "nasm")
	binaryFileName := tmpFolder + baseName
	err = exec.Command("ld", "-m", "elf_x86_64", "-static", "-o", binaryFileName, objectFileName).Run()
	checkError(err, "ld")
	cmd := exec.Command(binaryFileName)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	checkError(err)
	err = cmd.Wait()
	checkError(err, baseName)
}

func checkError(err error, s ...string) {
	if err != nil {
		if len(s) == 0 {
			fmt.Printf("err: %v\n", err)
		} else {
			fmt.Printf("%s: err: %v\n", s[0], err)
		}
		os.Exit(1)
	}
}
