package smbstatusreader

// Copyright 2021 by tobi@backfrak.de. All
// rights reserved. Use of this source code is governed
// by a BSD-style license that can be found in the
// LICENSE file.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tobi.backfrak.de/internal/commonbl"
)

// Type to represent a entry in the 'smbstatus -L -n' output table
type LockData struct {
	PID           int
	ClusterNodeId int // In case smaba is running in cluster mode, otherwise -1
	UserID        int
	DenyMode      string
	Access        string
	AccessMode    string
	Oplock        string
	SharePath     string
	Name          string
	Time          time.Time
}

// Implement Stringer Interface for LockData
func (lockData LockData) String() string {
	if lockData.ClusterNodeId > -1 {
		return fmt.Sprintf("ClusterNodeId: %d; PID: %d; UserID: %d; DenyMode: %s; Access: %s; AccessMode: %s; Oplock: %s; SharePath: %s; Name: %s: Time %s;",
			lockData.ClusterNodeId, lockData.PID, lockData.UserID, lockData.DenyMode, lockData.Access, lockData.AccessMode, lockData.Oplock,
			lockData.SharePath, lockData.Name, lockData.Time.Format(time.RFC3339))
	}
	return fmt.Sprintf("PID: %d; UserID: %d; DenyMode: %s; Access: %s; AccessMode: %s; Oplock: %s; SharePath: %s; Name: %s: Time %s;",
		lockData.PID, lockData.UserID, lockData.DenyMode, lockData.Access, lockData.AccessMode, lockData.Oplock,
		lockData.SharePath, lockData.Name, lockData.Time.Format(time.RFC3339))
}

// GetLockData - Get the entries out of the 'smbstatus -L -n' output table multiline string
// Will return an empty array if the data is in unexpected format
func GetLockData(data string, logger *commonbl.Logger) []LockData {
	var ret []LockData
	if strings.TrimSpace(data) == "No locked files" {
		return ret
	}

	lines := strings.Split(data, "\n")
	sepLineIndex := findSeperatorLineIndex(lines)

	if sepLineIndex < 1 {
		return ret
	}

	tableHeaderMatrix := getFieldMatrixFixLength(lines[sepLineIndex-1:sepLineIndex], "  ", 9)
	if len(tableHeaderMatrix) != 1 {
		return ret
	}
	tableHeaderFields := tableHeaderMatrix[0]

	if tableHeaderFields[0] != "Pid" || tableHeaderFields[5] != "Oplock" {
		return ret
	}

	i := -1
	for _, fields := range getFieldMatrix(lines[sepLineIndex+1:], " ") {
		i++
		var err error
		var entry LockData
		fieldLength := len(fields)
		if strings.Contains(fields[0], ":") {
			pidFields := strings.Split(fields[0], ":")
			entry.ClusterNodeId, err = strconv.Atoi(pidFields[0])
			if err != nil {
				logger.WriteErrorWithAddition(err, "while getting LockData ClusterNodeId")
				continue
			}
			entry.PID, err = strconv.Atoi(pidFields[1])
			if err != nil {
				logger.WriteErrorWithAddition(err, "while getting LockData PID (ClusterNodeId)")
				continue
			}
		} else {
			entry.ClusterNodeId = -1
			entry.PID, err = strconv.Atoi(fields[0])
			if err != nil {
				logger.WriteErrorWithAddition(err, "while getting LockData PID")
				continue
			}
		}
		entry.UserID, err = strconv.Atoi(fields[1])
		if err != nil {
			logger.WriteErrorWithAddition(err, "while getting LockData UserID")
			continue
		}
		entry.DenyMode = fields[2]
		entry.Access = fields[3]
		entry.AccessMode = fields[4]
		entry.Oplock = fields[5]
		entry.SharePath = fields[6]
		timeConvSuc := false
		var connectTime time.Time
		var lastNameIndex = -1
		timeConvSuc, connectTime = tryGetTimeStampFromStrArr(fields[fieldLength-5 : fieldLength])
		if timeConvSuc {
			entry.Time = connectTime
			lastNameIndex = fieldLength - 5
		} else {
			timeConvSuc, connectTime = tryGetTimeStampFromStrArr(fields[fieldLength-6 : fieldLength])
			if timeConvSuc {
				entry.Time = connectTime
				lastNameIndex = fieldLength - 6
			}
		}

		if lastNameIndex == -1 {
			logger.WriteErrorMessage(fmt.Sprintf("Not able to parse the time stamp in following LockData line: \"%s\"", lines[i]))
			continue
		}

		if lastNameIndex <= 7 {
			logger.WriteErrorMessage(fmt.Sprintf("Not able to find the name in following LockData line: \"%s\"", lines[i]))
			continue
		}

		name := ""
		for _, namePart := range fields[7:lastNameIndex] {
			name = fmt.Sprintf("%s %s", name, namePart)
		}
		entry.Name = strings.TrimSpace(name)

		ret = append(ret, entry)
	}
	return ret
}

// Type to represent a entry in the 'smbstatus -S -n' output table
type ShareData struct {
	Service       string
	PID           int
	ClusterNodeId int // In case smaba is running in cluster mode, otherwise -1
	Machine       string
	ConnectedAt   time.Time
	Encryption    string
	Signing       string
}

// Implement Stringer Interface for ShareData
func (shareData ShareData) String() string {
	if shareData.ClusterNodeId > -1 {
		return fmt.Sprintf("Service: %s; ClusterNodeId: %d; PID: %d; Machine: %s; ConnectedAt: %s; Encryption: %s; Signing: %s;",
			shareData.Service, shareData.ClusterNodeId, shareData.PID, shareData.Machine, shareData.ConnectedAt.Format(time.RFC3339),
			shareData.Encryption, shareData.Signing)
	}
	return fmt.Sprintf("Service: %s; PID: %d; Machine: %s; ConnectedAt: %s; Encryption: %s; Signing: %s;",
		shareData.Service, shareData.PID, shareData.Machine, shareData.ConnectedAt.Format(time.RFC3339),
		shareData.Encryption, shareData.Signing)
}

// GetShareData - Get the entries out of the 'smbstatus -S -n' output table multiline string
// Will return an empty array if the data is in unexpected format
func GetShareData(data string, logger *commonbl.Logger) []ShareData {
	var ret []ShareData
	lines := strings.Split(data, "\n")
	sepLineIndex := findSeperatorLineIndex(lines)

	if sepLineIndex < 1 {
		return ret
	}

	// Normal setup gives 6 fields in this line
	tableHeaderMatrix := getFieldMatrixFixLength(lines[sepLineIndex-1:sepLineIndex], "  ", 6)

	if len(tableHeaderMatrix) != 1 {
		// Cluster setup gives 7 fields in this line
		tableHeaderMatrix = getFieldMatrixFixLength(lines[sepLineIndex-1:sepLineIndex], "  ", 7)

		if len(tableHeaderMatrix) != 1 {
			return ret
		}
	}
	tableHeaderFields := tableHeaderMatrix[0]
	runningMode := "none"
	if tableHeaderFields[0] == "Service" && tableHeaderFields[3] == "Connected at" {
		runningMode = "normal"
	}

	if tableHeaderFields[0] == "PID" && tableHeaderFields[4] == "Protocol Version" {
		runningMode = "cluster"
	}

	if runningMode == "normal" {
		fieldMatrix := getFieldMatrixFixLength(lines[sepLineIndex+1:], " ", 12)
		if fieldMatrix != nil {
			for _, fields := range fieldMatrix {
				var err error
				var entry ShareData
				entry.Service = fields[0]
				if strings.Contains(fields[1], ":") {
					pidFields := strings.Split(fields[1], ":")
					entry.ClusterNodeId, err = strconv.Atoi(pidFields[0])
					if err != nil {
						logger.WriteErrorWithAddition(err, "while getting ShareData ClusterNodeId (normal - c12 - with :)")
						continue
					}
					entry.PID, err = strconv.Atoi(pidFields[1])
					if err != nil {
						logger.WriteErrorWithAddition(err, "while getting ShareData PID (normal - c12 - with :)")
						continue
					}
				} else {
					entry.ClusterNodeId = -1
					entry.PID, err = strconv.Atoi(fields[1])
					if err != nil {
						logger.WriteErrorWithAddition(err, "while getting ShareData PID (normal - c12 - without :)")
						continue
					}
				}
				entry.Machine = fields[2]
				timeStr := fmt.Sprintf("%s %s %s %s %s %s %s", fields[3], fields[4], fields[5], fields[6], fields[7], fields[8], fields[9])
				entry.ConnectedAt, err = time.Parse("Mon Jan 02 03:04:05 PM 2006 MST", timeStr)
				if err != nil {
					entry.ConnectedAt, err = time.Parse("Mon Jan 2 03:04:05 PM 2006 MST", timeStr)
					if err != nil {
						logger.WriteErrorWithAddition(err, "while getting ShareData ConnectedAt (normal - c12)")
						continue
					}
				}
				entry.Encryption = fields[10]
				entry.Signing = fields[11]

				ret = append(ret, entry)
			}
		} else {
			fieldMatrix = getFieldMatrixFixLength(lines[sepLineIndex+1:], " ", 11)
			if fieldMatrix != nil {
				for _, fields := range fieldMatrix {
					var err error
					var entry ShareData
					entry.Service = fields[0]
					if strings.Contains(fields[1], ":") {
						pidFields := strings.Split(fields[1], ":")
						entry.ClusterNodeId, err = strconv.Atoi(pidFields[0])
						if err != nil {
							logger.WriteErrorWithAddition(err, "while getting ShareData ClusterNodeId (normal - c11 - with :)")
							continue
						}
						entry.PID, err = strconv.Atoi(pidFields[1])
						if err != nil {
							logger.WriteErrorWithAddition(err, "while getting ShareData PID (normal - c11 - with :)")
							continue
						}
					} else {
						entry.ClusterNodeId = -1
						entry.PID, err = strconv.Atoi(fields[1])
						if err != nil {
							logger.WriteErrorWithAddition(err, "while getting ShareData PID (normal - c11 - without :)")
							continue
						}
					}
					entry.Machine = fields[2]
					timeStr := fmt.Sprintf("%s %s %s %s %s %s", fields[3], fields[4], fields[5], fields[6], fields[7], fields[8])
					entry.ConnectedAt, err = time.Parse("Mon Jan _2 15:04:05 2006 MST", timeStr)
					if err != nil {
						entry.ConnectedAt, err = time.Parse("Mo Jan _2 15:04:05 2006 MST", timeStr)
						if err != nil {
							logger.WriteErrorWithAddition(err, "while getting ShareData ConnectedAt (normal - c11)")
							continue
						}
					}
					entry.Encryption = fields[9]
					entry.Signing = fields[10]

					ret = append(ret, entry)
				}
			}
		}
	} else if runningMode == "cluster" {
		fieldMatrix := getFieldMatrixFixLength(lines[sepLineIndex+1:], " ", 8)
		if fieldMatrix != nil {
			for _, fields := range fieldMatrix {
				var err error
				var entry ShareData
				if strings.Contains(fields[0], ":") {
					pidFields := strings.Split(fields[0], ":")
					entry.ClusterNodeId, err = strconv.Atoi(pidFields[0])
					if err != nil {
						logger.WriteErrorWithAddition(err, "while getting ShareData ClusterNodeId (cluster - with :)")
						continue
					}
					entry.PID, err = strconv.Atoi(pidFields[1])
					if err != nil {
						logger.WriteErrorWithAddition(err, "while getting ShareData PID (cluster - with :)")
						continue
					}
				} else {
					entry.ClusterNodeId = -1
					entry.PID, err = strconv.Atoi(fields[0])
					if err != nil {
						logger.WriteErrorWithAddition(err, "while getting ShareData PID (cluster - without :)")
						continue
					}
				}
				entry.Machine = fmt.Sprintf("%s %s", fields[3], fields[4])
				entry.Encryption = fields[6]
				entry.Signing = fields[7]

				ret = append(ret, entry)
			}
		}
	}

	return ret
}

// Type to represent a entry in the 'smbstatus -p -n' output table
type ProcessData struct {
	PID             int
	ClusterNodeId   int // In case smaba is running in cluster mode, otherwise -1
	UserID          int
	GroupID         int
	Machine         string
	ProtocolVersion string
	Encryption      string
	Signing         string
	SambaVersion    string
}

// Implement Stringer Interface for ProcessData
func (processData ProcessData) String() string {
	if processData.ClusterNodeId > -1 {
		return fmt.Sprintf("ClusterNodeId: %d; PID: %d; UserID: %d; GroupID: %d; Machine: %s; ProtocolVersion: %s; Encryption: %s; Signing: %s;",
			processData.ClusterNodeId, processData.PID, processData.UserID, processData.GroupID, processData.Machine, processData.ProtocolVersion,
			processData.Encryption, processData.Signing)
	}
	return fmt.Sprintf("PID: %d; UserID: %d; GroupID: %d; Machine: %s; ProtocolVersion: %s; Encryption: %s; Signing: %s;",
		processData.PID, processData.UserID, processData.GroupID, processData.Machine, processData.ProtocolVersion,
		processData.Encryption, processData.Signing)
}

// GetProcessData - Get the entries out of the 'smbstatus -p -n' output table multiline string
// Will return an empty array if the data is in unexpected format
func GetProcessData(data string, logger *commonbl.Logger) []ProcessData {
	var ret []ProcessData
	lines := strings.Split(data, "\n")
	sepLineIndex := findSeperatorLineIndex(lines)

	if sepLineIndex < 2 {
		return ret
	}

	var sambaVersion string
	sambaVersionLine := lines[sepLineIndex-2 : sepLineIndex-1][0]
	if strings.HasPrefix(sambaVersionLine, "Samba version") {
		sambaVersion = strings.TrimSpace(strings.Replace(sambaVersionLine, "Samba version", "", 1))
	} else {
		return ret
	}

	tableHeaderMatrix := getFieldMatrixFixLength(lines[sepLineIndex-1:sepLineIndex], "  ", 7)
	if len(tableHeaderMatrix) != 1 {
		return ret
	}
	tableHeaderFields := tableHeaderMatrix[0]

	if tableHeaderFields[1] != "Username" || tableHeaderFields[4] != "Protocol Version" {
		return ret
	}

	for _, fields := range getFieldMatrixFixLength(lines[sepLineIndex+1:], " ", 8) {
		var err error
		var entry ProcessData
		// In cluster versions samba adds an extra id separated by ':'
		if strings.Contains(fields[0], ":") {
			pidFields := strings.Split(fields[0], ":")
			entry.ClusterNodeId, err = strconv.Atoi(pidFields[0])
			if err != nil {
				logger.WriteErrorWithAddition(err, "while getting ProcessData ClusterNodeId")
				continue
			}
			entry.PID, err = strconv.Atoi(pidFields[1])
			if err != nil {
				logger.WriteErrorWithAddition(err, "while getting ProcessData PID (with :)")
				continue
			}
		} else {
			entry.ClusterNodeId = -1
			entry.PID, err = strconv.Atoi(fields[0])
			if err != nil {
				logger.WriteErrorWithAddition(err, "while getting ProcessData PID (without :)")
				continue
			}
		}
		// In cluster versions samba does not print the users id, but nobody
		if fields[1] == "nobody" {
			entry.UserID = -1
		} else {
			entry.UserID, err = strconv.Atoi(fields[1])
			if err != nil {
				logger.WriteErrorWithAddition(err, "while getting ProcessData UserID")
				continue
			}
		}
		// In cluster versions samba does not print the group id, but nogroup
		if fields[2] == "nogroup" {
			entry.GroupID = -1
		} else {
			entry.GroupID, err = strconv.Atoi(fields[2])
			if err != nil {
				logger.WriteErrorWithAddition(err, "while getting ProcessData GroupID")
				continue
			}
		}
		entry.Machine = fmt.Sprintf("%s %s", fields[3], fields[4])
		entry.ProtocolVersion = fields[5]
		entry.Encryption = fields[6]
		entry.Signing = fields[7]
		entry.SambaVersion = sambaVersion

		ret = append(ret, entry)
	}
	return ret
}

func GetPsData(data string, logger *commonbl.Logger) []commonbl.PsUtilPidData {
	var ret []commonbl.PsUtilPidData
	errConv := json.Unmarshal([]byte(data), &ret)
	if errConv != nil {
		logger.WriteErrorWithAddition(errConv, "while converting PsData json")
		return []commonbl.PsUtilPidData{}
	}

	return ret
}

func getFieldMatrixFixLength(dataLines []string, separator string, lineFields int) [][]string {

	var fieldMatrix [][]string

	for _, matrixLine := range getFieldMatrix(dataLines, separator) {
		if len(matrixLine) == lineFields {
			fieldMatrix = append(fieldMatrix, matrixLine)
		}
	}

	return fieldMatrix
}

func getFieldMatrix(dataLines []string, separator string) [][]string {

	var fieldMatrix [][]string

	for _, line := range dataLines {
		fields := strings.Split(line, separator)
		var matrixLine []string
		for _, field := range fields {
			trimmedField := strings.TrimSpace(field)
			if trimmedField != "" {
				matrixLine = append(matrixLine, trimmedField)
			}
		}
		fieldMatrix = append(fieldMatrix, matrixLine)
	}

	return fieldMatrix
}

func tryGetTimeStampFromStrArr(fields []string) (bool, time.Time) {
	timeStr := ""
	var ret time.Time
	var err error
	for _, sec := range fields {
		timeStr = fmt.Sprintf("%s %s", timeStr, sec)
	}
	timeStr = strings.TrimSpace(timeStr)
	ret, err = time.ParseInLocation(time.ANSIC, timeStr, time.Now().Location())
	if err == nil {
		return true, ret
	}
	ret, err = time.Parse(time.ANSIC, timeStr)
	if err == nil {
		return true, ret
	}
	ret, err = time.Parse("Mon Jan 02 03:04:05 PM 2006 MST", timeStr)
	if err == nil {
		return true, ret
	}
	ret, err = time.Parse("Mon Jan 2 03:04:05 PM 2006 MST", timeStr)
	if err == nil {
		return true, ret
	}
	ret, err = time.Parse("Mon Jan _2 15:04:05 2006 MST", timeStr)
	if err == nil {
		return true, ret
	}
	ret, err = time.Parse("Mo Jan _2 15:04:05 2006 MST", timeStr)
	if err == nil {
		return true, ret
	}

	return false, time.Now()
}

func findSeperatorLineIndex(lines []string) int {

	for i, line := range lines {
		if strings.HasPrefix(line, "-----------------------------------------") {
			return i
		}
	}

	return -1
}
