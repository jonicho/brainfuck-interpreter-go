package main

import (
	"bufio"
	"fmt"
	"os"
)

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

func parseProgram(programScanner *bufio.Scanner) *instruction {
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
			nextInstruction.loop = parseProgram(programScanner)
		case "]":
			nextInstruction = nil
		}
		currentInstruction.next = nextInstruction
		currentInstruction = nextInstruction
	}
	return firstInstruction
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
	programScanner := bufio.NewScanner(programFile)
	programScanner.Split(bufio.ScanRunes)
	parseProgram(programScanner)
}
