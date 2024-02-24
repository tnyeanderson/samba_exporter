package commonbl

// Copyright 2021 by tobi@backfrak.de. All
// rights reserved. Use of this source code is governed
// by a BSD-style license that can be found in the
// LICENSE file.

import (
	"fmt"
	"os"
)

// ConsoleLogger - A "class" with log functions
type ConsoleLogger struct {
	Verbose bool
}

// Get a new instance of the Logger
func NewConsoleLogger(verbose bool) *ConsoleLogger {
	ret := ConsoleLogger{verbose}

	return &ret
}

// WriteInformation - Write a Info message to Stdout, will be prefixed with "Information: "
func (logger *ConsoleLogger) WriteInformation(message string) {
	fmt.Fprintln(os.Stdout, fmt.Sprintf("Information: %s", message))

	return
}

// WriteVerbose - Write a Verbose message to Stdout. Message will be written only if logger.Verbose is true.
// The message will be prefixed with "Verbose :"
func (logger *ConsoleLogger) WriteVerbose(message string) {
	if logger.Verbose {
		fmt.Fprintln(os.Stdout, fmt.Sprintf("Verbose: %s", message))
	}

	return
}

// WriteErrorMessage - Write the message to Stderr. The Message will be prefixed with "Error: "
func (logger *ConsoleLogger) WriteErrorMessage(message string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf("Error: %s", message))
}

// WriteError - Writes the err.Error() output to Stderr
func (logger *ConsoleLogger) WriteError(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
}

// WriteError - Writes the 'err.Error() - addition' output to Stderr
func (logger *ConsoleLogger) WriteErrorWithAddition(err error, addition string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf("%s - %s", err.Error(), addition))
}
