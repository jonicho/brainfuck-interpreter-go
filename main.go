package main

import (
	"bufio"
	"fmt"
	"os"
)

var programScanner *bufio.Scanner
var data []byte
var dataPtr int
var stdinReader *bufio.Reader
var program *instruction

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
	value           int
	offset          int
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
			nextInstruction.value = -1
			fallthrough
		case ">":
			nextInstruction.instructionType = move
		case "-":
			nextInstruction.value = -1
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

func increaseDataArraySize(minRequiredIndex int) {
	newSize := len(data)
	for newSize < minRequiredIndex+1 {
		newSize *= 2
	}
	data = append(data, make([]byte, newSize-len(data))...)
}

func runInstruction(instruction *instruction) {
	for instruction != nil {
		switch instruction.instructionType {
		case nop:
		case move:
			dataPtr += instruction.value
		case add:
			if dataPtr+instruction.offset >= len(data) {
				increaseDataArraySize(dataPtr + instruction.offset)
			}
			data[dataPtr+instruction.offset] += byte(instruction.value)
		case print:
			if dataPtr+instruction.offset >= len(data) {
				increaseDataArraySize(dataPtr + instruction.offset)
			}
			fmt.Printf("%c", data[dataPtr+instruction.offset])
		case read:
			if dataPtr+instruction.offset >= len(data) {
				increaseDataArraySize(dataPtr + instruction.offset)
			}
			data[dataPtr+instruction.offset], _ = stdinReader.ReadByte()
		case loop:
			for data[dataPtr] != 0 {
				runInstruction(instruction.loop)
			}
		case clear:
			if dataPtr+instruction.offset >= len(data) {
				increaseDataArraySize(dataPtr + instruction.offset)
			}
			data[dataPtr+instruction.offset] = 0
		}

		if dataPtr >= len(data) {
			increaseDataArraySize(dataPtr)
		}
		instruction = instruction.next
	}
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

func optimizeProgram() {
	program = removeNops(program)
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("Usage: %s file\n", args[0])
		os.Exit(0)
	}

	programFile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Failed to open file!\n")
		os.Exit(1)
	}
	programScanner = bufio.NewScanner(programFile)
	programScanner.Split(bufio.ScanRunes)
	program = parseProgram()

	optimizeProgram()

	data = make([]byte, 1)
	stdinReader = bufio.NewReader(os.Stdin)
	runInstruction(program)
}
