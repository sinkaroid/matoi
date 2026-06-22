package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func parseStat(pid int, parentMap map[int]int) {
	statBytes, statErr := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if statErr != nil {
		return
	}
	statStr := string(statBytes)
	rParen := strings.LastIndex(statStr, ")")
	if rParen == -1 || len(statStr) <= rParen+2 {
		return
	}
	fieldsAfter := strings.Fields(statStr[rParen+2:])
	if len(fieldsAfter) >= 2 {
		if ppid, errParse := strconv.Atoi(fieldsAfter[1]); errParse == nil {
			parentMap[pid] = ppid
		}
	}
}

func parseStatm(pid int, rssMap map[int]int64, pageSize int64) {
	statmBytes, statmErr := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if statmErr != nil {
		return
	}
	parts := strings.Fields(string(statmBytes))
	if len(parts) >= 2 {
		if rssPages, errParse := strconv.ParseInt(parts[1], 10, 64); errParse == nil {
			rssMap[pid] = rssPages * pageSize
		}
	}
}

func processDir(f os.DirEntry, parentMap map[int]int, rssMap map[int]int64, pageSize int64) int {
	pid, err := strconv.Atoi(f.Name())
	if err != nil {
		return 0
	}

	parseStat(pid, parentMap)
	parseStatm(pid, rssMap, pageSize)

	cmdBytes, readErr := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if readErr == nil && strings.Contains(string(cmdBytes), "flaresolverr.py") {
		return pid
	}
	return 0
}

func calculateDescendantRSS(flaresolverrPID int, parentMap map[int]int, rssMap map[int]int64) int64 {
	var totalRSS int64
	var isDescendant func(pid int) bool
	isDescendant = func(pid int) bool {
		if pid == flaresolverrPID {
			return true
		}
		ppid, exists := parentMap[pid]
		if !exists || ppid == 0 {
			return false
		}
		return isDescendant(ppid)
	}

	for pid, rss := range rssMap {
		if isDescendant(pid) {
			totalRSS += rss
		}
	}
	return totalRSS
}

func getMemory() (int64, string) {
	files, err := os.ReadDir("/proc")
	if err != nil {
		return 0, "error"
	}

	pageSize := int64(os.Getpagesize())
	var flaresolverrPID int

	parentMap := make(map[int]int)
	rssMap := make(map[int]int64)

	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		if pid := processDir(f, parentMap, rssMap, pageSize); pid != 0 {
			flaresolverrPID = pid
		}
	}

	if flaresolverrPID == 0 {
		return 0, "flaresolverr not found"
	}

	totalRSS := calculateDescendantRSS(flaresolverrPID, parentMap, rssMap)
	if totalRSS > 0 {
		return totalRSS, "procfs: strict process tree (flaresolverr + children)"
	}

	return 0, "unknown"
}

func getLimit() int64 {
	data, err := os.ReadFile("/sys/fs/cgroup/memory.max")
	if err == nil {
		str := strings.TrimSpace(string(data))
		if str != "max" {
			limit, errParse := strconv.ParseInt(str, 10, 64)
			if errParse == nil {
				return limit
			}
		}
	}
	data, err = os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	if err == nil {
		limit, errParse := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		if errParse == nil {
			return limit
		}
	}
	return 0
}
