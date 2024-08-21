package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	major, minor, patch, commitsAhead int
	preRelease, meta                  string
	prefix, dirty                     bool

	// Flags
	showDirty = true
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelError)
	var err error
	tag, tagErr := execute("git", "describe", "--tags", "--abbrev=0")
	rawDescribe, describeErr := execute("git", "describe", "--tags", "--long", "--dirty")

	if tagErr != nil || describeErr != nil {
		slog.Warn("No tags or commits found, probably in a new repo – setting describe/tag to v0.0.0")
		prefix = true
		dirty = true
		tag = "v0.0.0"
		rawDescribe = "v0.0.0-0-g0000000-dirty"
	} else {
		describe := strings.Split(rawDescribe[len(tag)+1:], "-")
		slog.Debug("Has found a tag!", "describe", describe, "rawDescribe", rawDescribe, "tag", tag)
		commitsAhead, err = strconv.Atoi(describe[0])
		if err != nil {
			slog.Error("Exiting: Failed to convert commitsAhead to int", "error", err, "describe", describe, "rawDescribe", rawDescribe)
			os.Exit(3)
		}
		if len(describe) > 2 {
			dirty = true
		}
	}

	var version string
	if strings.HasPrefix(tag, "v") {
		version = tag[1:]
		prefix = true
	}

	if strings.Contains(version, "+") {
		split := strings.Split(version, "+")
		version = split[0]
		meta = split[1]
	} /*else { TODO: No clue was this was supposed to do
		meta = hash
	}*/

	if strings.Contains(version, "-") {
		split := strings.SplitN(version, "-", 2)
		version = split[0]
		preRelease = split[1]
	}

	if version == "" {
		major = 0
		minor = 0
		patch = 0
	} else {
		split := strings.Split(version, ".")
		if len(split) != 3 {
			slog.Error("Exiting: Failed to split version into major, minor and patch", "version", version)
			os.Exit(4)
		}
		major, err = strconv.Atoi(split[0])
		if err != nil {
			slog.Error("Exiting: Failed to convert major to int", "error", err, "version", version)
			os.Exit(5)
		}
		minor, err = strconv.Atoi(split[1])
		if err != nil {
			slog.Error("Exiting: Failed to convert minor to int", "error", err, "version", version)
			os.Exit(6)
		}
		patch, err = strconv.Atoi(split[2])
		if err != nil {
			slog.Error("Exiting: Failed to convert patch to int", "error", err, "version", version)
			os.Exit(7)
		}
	}

	var builder strings.Builder
	if prefix {
		builder.WriteString("v")
	}
	builder.WriteString(strconv.Itoa(major))
	builder.WriteString(".")
	builder.WriteString(strconv.Itoa(minor))
	builder.WriteString(".")
	if (commitsAhead > 0 || dirty) && preRelease == "" {
		patch++
	}
	builder.WriteString(strconv.Itoa(patch))
	//if (patch == 0 && commitsAhead == 0) || (patch != 0 && commitsAhead == 0) {

	// Write prerelease if exists or otherwise dev and the amounts of commits we're ahead
	if preRelease != "" || commitsAhead != 0 {
		builder.WriteString("-")

		if preRelease == "" {
			builder.WriteString("dev.")
			builder.WriteString(strconv.Itoa(commitsAhead))
		} else if preRelease != "" && commitsAhead == 0 {
			builder.WriteString(preRelease)
		} else if preRelease != "" && commitsAhead != 0 {
			builder.WriteString(preRelease)
			builder.WriteString(".")
			builder.WriteString(strconv.Itoa(commitsAhead))
		}
	}

	if meta != "" || dirty {
		builder.WriteString("+")
		if meta != "" {
			builder.WriteString(meta)
		}
		if dirty && meta != "" {
			builder.WriteString(".dirty")
		} else if dirty && meta == "" {
			builder.WriteString("dirty")
		}
	}

	version = builder.String()

	slog.Debug(fmt.Sprintf("Set version to %s according to raw tag %s", version, rawDescribe), "major", major, "minor", minor, "patch", patch, "preRelease", preRelease, "commitsAhead", commitsAhead, "meta", meta, "prefix", prefix, "dirty", dirty)

	// Print actual version to stdout
	fmt.Println(version)
}

// execute runs the given command and returns the output as a string.
// It will perform some error handling that is very specific to the git command.
// If it detects the git command failing due to not being in a git repository, it will exit the program.
// If it detects the git command failing due to no tags or commits, it will return the error.
// For any other errors, the program will exit, or if no error, it will return the output.
// This also means, that if an error is returned, there is an repository, but no tags and/or commits yet.
func execute(cmd string, args ...string) (string, *exec.ExitError) {
	buffer, err := exec.Command(cmd, args...).Output()
	output := strings.TrimSpace(string(buffer))

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			panic(err) // got an error, but not of type ExtiError?
		}
		stderr := string(exitErr.Stderr)

		if strings.HasPrefix(stderr, "fatal: not a git repository") {
			slog.Error("Exiting: You must be in a git repository")
			os.Exit(1)
		}

		if strings.HasPrefix(stderr, "fatal: No names found") {
			return output, exitErr
		}

		slog.Error("Exiting: An unexpected error occurred when running git describe", "exitCode", exitErr.ExitCode(), "stderr", stderr)
		os.Exit(2)
	}

	return output, nil
}
